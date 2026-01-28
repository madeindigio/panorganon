# Plan de Implementación: Sistema de Hooks con Filtros Lua en Panorganon

**Fecha**: 2026-01-28
**Proyecto**: Panorganon MCP Server
**Versión**: 1.0

---

## Resumen Ejecutivo

Este documento presenta el plan completo para implementar un sistema de hooks con filtros Lua en Panorganon, permitiendo interceptar y modificar los parámetros de entrada y salida de las llamadas a herramientas MCP downstream. El objetivo es proporcionar una capa de seguridad, privacidad y transformación configurable mediante scripts Lua.

## Objetivos del Proyecto

1. **Seguridad y Privacidad**: Prevenir la exposición de datos sensibles (API keys, secrets, paths del sistema)
2. **Transformación de Datos**: Modificar inputs/outputs según reglas de negocio
3. **Auditoría**: Registrar y auditar todas las llamadas MCP con posibilidad de filtrado
4. **Flexibilidad**: Permitir configuración dinámica mediante scripts Lua sin recompilar
5. **Performance**: Minimizar overhead en el flujo de ejecución normal

## Arquitectura Propuesta

### Diagrama de Flujo con Hooks

```
┌─────────────────────────────────────────────────────────────┐
│ Cliente MCP (Claude, etc.)                                  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Panorganon: Handler exec_tool                               │
│ (internal/server/handler.go)                                │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ ExecutorService.ExecTool()                                  │
│ (internal/tools/executor.go)                                │
│                                                              │
│  1. lookupTool() → ToolRecord                               │
│  2. validateParameters()                                    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🔵 HOOK POINT 1: INPUT FILTER                       │  │
│  │ filterManager.ApplyInputFilter()                     │  │
│  │ → Ejecuta función Lua: <servername>-input           │  │
│  │ → Modifica parámetros antes de enviar               │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  3. executeTool()                                           │
│     ├─ manager.GetOrStart(serverName)                       │
│     ├─ client.CallTool(toolName, filteredParams)            │
│     │                                                        │
│     │  ┌───────────────────────────────────────────────┐   │
│     │  │ 🔵 HOOK POINT 2: OUTPUT FILTER               │   │
│     │  │ filterManager.ApplyOutputFilter()             │   │
│     │  │ → Ejecuta función Lua: <servername>-output    │   │
│     │  │ → Modifica resultado antes de devolver        │   │
│     │  └───────────────────────────────────────────────┘   │
│     │                                                        │
│     └─ manager.Stop() si no keepalive                       │
│                                                              │
│  4. Retorna ExecutionResult                                 │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Downstream MCP Server (remembrances-mcp, hyper-mcp, etc.)  │
└─────────────────────────────────────────────────────────────┘
```

### Componentes Nuevos

1. **FilterManager** (`internal/luafilters/manager.go`):
   - Gestiona el estado de Lua
   - Carga y ejecuta scripts de filtros
   - Coordina aplicación de filtros input/output

2. **LuaState Initialization** (`internal/luafilters/lua.go`):
   - Inicializa gopher-lua con todos los plugins
   - Registra módulos: strings, http, env, fs, template, yaml, sh, time, re, json

3. **Helpers Go-Lua** (`internal/luafilters/helpers.go`):
   - Conversión bidireccional de tipos Go ↔ Lua
   - Funciones de utilidad para manipular tablas Lua

4. **Types and Interfaces** (`internal/luafilters/types.go`):
   - HookContext: Contexto de ejecución
   - HookResult: Resultado de aplicar filtros
   - FilterFunc: Interfaz de función de filtro

### Configuración YAML

Extensión del archivo `config.yaml` con nueva sección:

```yaml
filters:
  enabled: true
  script_path: "./filters/panorganon-filters.lua"
  timeout: 5s
  strict_mode: false
  hot_reload: false
  log_filtered_data: false
```

### Convención de Nombres de Funciones Lua

Los scripts Lua deben definir funciones con el formato:

- **Input Filter**: `<server-name>-input`
- **Output Filter**: `<server-name>-output`

Ejemplos:
- `remembrances-mcp-input(context)` → Filtro de entrada para remembrances-mcp
- `remembrances-mcp-output(context)` → Filtro de salida para remembrances-mcp
- `hyper-mcp-input(context)` → Filtro de entrada para hyper-mcp

Si la función no existe, no se aplica ningún filtro.

---

## Plan de Desarrollo por Fases

El desarrollo se divide en **6 fases iterativas** que permiten validar cada componente antes de avanzar.

### 📦 FASE 1: Setup de Gopher-Lua y Plugins

**Duración Estimada**: 2-3 días
**Referencia**: Fact `fase1_setup_lua` en remembrances

**Objetivos**:
- Instalar y configurar gopherlua con todos los plugins necesarios
- Crear estructura base del paquete luafilters
- Implementar funciones de interoperabilidad Go-Lua

**Entregables**:
- ✅ Dependencias instaladas en go.mod
- ✅ Estructura de directorios internal/luafilters/
- ✅ InitLuaState() funcional con módulos cargados
- ✅ Funciones helper de conversión Go ↔ Lua

**Validación**:
- Test de inicialización de Lua sin errores
- Test de carga de cada módulo (strings, http, yaml, etc.)
- Test de conversión bidireccional de tipos básicos

---

### 🔌 FASE 2: Sistema de Hooks - Interfaces y Tipos

**Duración Estimada**: 2-3 días
**Referencia**: Fact `fase2_sistema_hooks` en remembrances

**Objetivos**:
- Definir interfaces y tipos para el sistema de hooks
- Implementar FilterManager con carga de scripts
- Crear lógica de ejecución de funciones Lua por convención de nombres

**Entregables**:
- ✅ Tipos: HookContext, HookResult, FilterFunc
- ✅ FilterManager con métodos:
  - NewFilterManager(scriptPath)
  - LoadScript()
  - ApplyInputFilter()
  - ApplyOutputFilter()
  - ReloadScript()
- ✅ Manejo de errores y timeouts

**Validación**:
- Test de carga de script Lua válido
- Test de ejecución de función Lua desde Go
- Test de timeout en ejecución de filtros
- Test de manejo de errores en scripts malformados

---

### ⚙️ FASE 3: Configuración YAML para Hooks

**Duración Estimada**: 1-2 días
**Referencia**: Fact `fase3_config_yaml` en remembrances

**Objetivos**:
- Extender configuración YAML con sección "filters"
- Implementar validación de configuración
- Crear archivo de ejemplo de filtros Lua

**Entregables**:
- ✅ Estructura FiltersConfig en config.go
- ✅ Parsing y validación con Viper
- ✅ examples/config.example.yaml actualizado
- ✅ examples/filters.example.lua con ejemplos básicos

**Validación**:
- Test de parsing de configuración con sección filters
- Test de valores por defecto si sección no existe
- Test de validación de script_path inexistente
- Script de ejemplo ejecuta sin errores

---

### 📥 FASE 4: Integración de Hooks de Input (Pre-Ejecución)

**Duración Estimada**: 2-3 días
**Referencia**: Fact `fase4_hooks_input` en remembrances

**Objetivos**:
- Interceptar parámetros antes de enviar a servidor MCP downstream
- Aplicar filtros de input configurados en Lua
- Implementar logging y auditoría de modificaciones

**Entregables**:
- ✅ Modificación de ExecutorService con campo filterManager
- ✅ Hook insertado en ExecTool() después de validateParameters
- ✅ Implementación completa de ApplyInputFilter()
- ✅ Funciones Lua helper: redact_field, remove_field, validate_field, add_field

**Validación**:
- Test de ejecución con filtro input que modifica parámetros
- Test de modo strict con filtro que falla
- Test de fallback si filtro no existe
- Test de logging de datos filtrados

**Puntos de Integración**:
- Archivo: `internal/tools/executor.go`
- Línea aproximada: 80 (después de validateParameters)
- Contexto pasado a Lua: server_name, tool_name, parameters, request_id, timestamp

---

### 📤 FASE 5: Integración de Hooks de Output (Post-Ejecución)

**Duración Estimada**: 2-3 días
**Referencia**: Fact `fase5_hooks_output` en remembrances

**Objetivos**:
- Interceptar resultados antes de devolverlos al cliente
- Aplicar filtros de output para redactar información sensible
- Transformar o enriquecer datos de salida

**Entregables**:
- ✅ Hook insertado en executeTool() después de CallTool
- ✅ Funciones de conversión: resultToMap(), mapToResult()
- ✅ Implementación completa de ApplyOutputFilter()
- ✅ Funciones Lua helper: redact_in_content, filter_content_type, add_metadata, transform_content

**Validación**:
- Test de ejecución con filtro output que modifica resultado
- Test de redacción de API keys en respuestas
- Test de transformación de contenido
- Test de preservación de estructura mcp.CallToolResult

**Puntos de Integración**:
- Archivo: `internal/tools/executor.go`
- Método: `executeTool()`
- Línea aproximada: 275 (después de CallTool)
- Contexto pasado a Lua: server_name, tool_name, result, duration, request_id

**Casos de Uso Específicos**:
- Redactar API keys y tokens en respuestas
- Filtrar paths absolutos del sistema
- Eliminar información sensible (IPs, emails)
- Agregar watermarks o metadatos de auditoría

---

### ✅ FASE 6: Testing, Documentación y Ejemplos

**Duración Estimada**: 3-4 días
**Referencia**: Fact `fase6_testing_docs` en remembrances

**Objetivos**:
- Crear suite completa de tests unitarios e integración
- Documentar API Lua y guías de uso
- Proporcionar ejemplos prácticos de filtros comunes

**Entregables**:
- ✅ Tests unitarios (cobertura > 80%):
  - internal/luafilters/*_test.go
  - internal/tools/executor_test.go (con filtros)
- ✅ Tests de integración end-to-end
- ✅ Benchmarks de performance
- ✅ Documentación completa (docs/lua-filters.md)
- ✅ Ejemplos de filtros prácticos:
  - privacy-filter.lua
  - security-filter.lua
  - logging-filter.lua
  - rate-limit-filter.lua
  - transformation-filter.lua
- ✅ README.md actualizado con sección de filtros
- ✅ Pipeline CI/CD con tests de filtros

**Validación**:
- Todos los tests pasan
- Cobertura de código > 80% en paquete luafilters
- Benchmarks muestran overhead < 10ms por filtro
- Documentación revisada y completa
- Ejemplos ejecutan correctamente

---

## Dependencias Técnicas

### Librerías Go a Instalar

```bash
go get github.com/yuin/gopher-lua
go get github.com/vadv/gopher-lua-libs/strings
go get github.com/vadv/gopher-lua-libs/http
go get github.com/vadv/gopher-lua-libs/env
go get github.com/vadv/gopher-lua-libs/time
go get github.com/vadv/gopher-lua-libs/sh
go get github.com/vadv/gopher-lua-libs/re
go get github.com/vadv/gopher-lua-libs/yaml
go get github.com/vadv/gopher-lua-libs/template
go get github.com/layeh/gopher-json
```

Nota: Para `fs`, verificar disponibilidad en vadv/gopher-lua-libs o buscar alternativa compatible.

### Referencia de Implementación

Proyecto **essh** de sevir:
- URL: https://github.com/sevir/essh
- Archivo clave: `essh/lualib.go`
- Proporciona patrones probados de integración gopherlua + plugins

---

## API Lua para Filtros

### Contexto Pasado a Funciones Lua

#### Input Filter Context
```lua
function remembrances-mcp-input(context)
    -- context contiene:
    -- context.server_name: string
    -- context.tool_name: string
    -- context.parameters: table (parámetros originales)
    -- context.request_id: string
    -- context.timestamp: number (Unix timestamp)
    
    -- Debe retornar: table con parámetros modificados
end
```

#### Output Filter Context
```lua
function remembrances-mcp-output(context)
    -- context contiene:
    -- context.server_name: string
    -- context.tool_name: string
    -- context.result: table (resultado MCP)
    -- context.duration: number (milisegundos)
    -- context.request_id: string
    
    -- Debe retornar: table con resultado modificado
end
```

### Funciones Helper Disponibles

```lua
-- Redactar valor de un campo
redact_field(params, "api_key")  --> params.api_key = "[REDACTED]"

-- Eliminar campo completamente
remove_field(params, "sensitive_data")

-- Validar campo con regex
if not validate_field(params, "email", "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$") then
    error("Invalid email format")
end

-- Agregar campo
add_field(params, "audit_timestamp", os.time())

-- Redactar en contenido de texto (para output)
redact_in_content(result, "sk-[a-zA-Z0-9]+")  --> Redacta API keys OpenAI

-- Filtrar items de content por tipo
filter_content_type(result, "image")  --> Elimina contenido de tipo imagen

-- Agregar metadata
add_metadata(result, "filtered_by", "panorganon-lua-filter")
```

### Módulos Lua Disponibles

Los siguientes módulos están precargados y disponibles en scripts:

- **strings**: Manipulación avanzada de strings
- **http**: Llamadas HTTP (con precaución)
- **env**: Acceso a variables de entorno
- **fs**: Operaciones de filesystem (lectura)
- **template**: Templates de texto
- **yaml**: Parsing y generación de YAML
- **sh**: Ejecución de comandos shell (deshabilitado por seguridad)
- **time**: Manipulación de tiempo y fechas
- **re**: Expresiones regulares
- **json**: Parsing y generación de JSON

---

## Ejemplo de Script Lua Completo

```lua
-- panorganon-filters.lua
-- Ejemplo de filtros para diferentes servidores MCP

-- ============================================
-- Filtros para remembrances-mcp
-- ============================================

function remembrances-mcp-input(context)
    local params = context.parameters
    
    -- Prevenir búsquedas de información sensible
    if params.query then
        -- Bloquear búsquedas de passwords o keys
        if string.match(params.query, "password") or 
           string.match(params.query, "api[_-]?key") then
            error("Búsqueda de información sensible bloqueada")
        end
    end
    
    -- Limitar tamaño de contenido almacenado
    if params.content and #params.content > 10000 then
        params.content = string.sub(params.content, 1, 10000) .. "... [truncated]"
    end
    
    -- Agregar metadata de auditoría
    params._audit = {
        timestamp = os.time(),
        server = context.server_name,
        tool = context.tool_name
    }
    
    return params
end

function remembrances-mcp-output(context)
    local result = context.result
    
    -- Redactar paths absolutos del sistema
    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                -- Redactar paths como /home/user/...
                item.text = string.gsub(item.text, "/home/[^/]+", "/home/[USER]")
                item.text = string.gsub(item.text, "/Users/[^/]+", "/Users/[USER]")
            end
        end
    end
    
    -- Agregar watermark
    if result.content then
        table.insert(result.content, {
            type = "text",
            text = "\n[Filtered by Panorganon Lua Filters]"
        })
    end
    
    return result
end

-- ============================================
-- Filtros para hyper-mcp
-- ============================================

function hyper-mcp-input(context)
    local params = context.parameters
    
    -- Validar URLs para prevenir SSRF
    if params.url then
        -- Bloquear localhost, IPs privadas
        if string.match(params.url, "localhost") or
           string.match(params.url, "127%.0%.0%.1") or
           string.match(params.url, "192%.168%.") or
           string.match(params.url, "10%.%d+%.") then
            error("URL a red privada bloqueada")
        end
    end
    
    return params
end

function hyper-mcp-output(context)
    local result = context.result
    
    -- Redactar API keys en respuestas (OpenAI, Anthropic, etc.)
    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                -- Redactar keys de OpenAI
                item.text = string.gsub(item.text, "sk%-[a-zA-Z0-9]+", "sk-[REDACTED]")
                -- Redactar keys de Anthropic
                item.text = string.gsub(item.text, "sk%-ant%-[a-zA-Z0-9%-]+", "sk-ant-[REDACTED]")
            end
        end
    end
    
    return result
end

-- ============================================
-- Filtro global (fallback)
-- ============================================

function global-input(context)
    -- Se ejecuta si no existe filtro específico del servidor
    print(string.format("[AUDIT] Tool: %s, Server: %s", 
        context.tool_name, context.server_name))
    return context.parameters
end
```

---

## Riesgos y Mitigaciones

### Riesgos Identificados

1. **Performance Overhead**
   - **Mitigación**: Benchmarks en fase 6, timeout configurable, caché de scripts compilados

2. **Errores en Scripts Lua**
   - **Mitigación**: Modo strict opcional, fallback a parámetros originales, logging exhaustivo

3. **Seguridad de Scripts**
   - **Mitigación**: Sandbox de Lua, deshabilitar módulos peligrosos (sh), validación de scripts

4. **Complejidad de Debugging**
   - **Mitigación**: Logging detallado, modo log_filtered_data, ejemplos documentados

5. **Mantenimiento de Scripts**
   - **Mitigación**: Versionado de scripts, tests de regresión, hot-reload opcional

---

## Métricas de Éxito

1. **Funcionalidad**:
   - ✅ Input filters funcionan en 100% de casos de prueba
   - ✅ Output filters funcionan en 100% de casos de prueba
   - ✅ Manejo de errores sin interrumpir flujo normal

2. **Performance**:
   - ✅ Overhead de filtros < 10ms por llamada
   - ✅ No memory leaks en ejecución prolongada
   - ✅ Scripts se cargan en < 100ms

3. **Calidad**:
   - ✅ Cobertura de tests > 80%
   - ✅ Documentación completa y ejemplos funcionales
   - ✅ Sin regresiones en funcionalidad existente

4. **Seguridad**:
   - ✅ API keys redactadas en logs y respuestas
   - ✅ Paths del sistema filtrados correctamente
   - ✅ Sandbox de Lua sin brechas de seguridad

---

## Cronograma Estimado

| Fase | Duración | Dependencias | Entregables Clave |
|------|----------|--------------|-------------------|
| Fase 1 | 2-3 días | - | Setup Lua + plugins |
| Fase 2 | 2-3 días | Fase 1 | FilterManager funcional |
| Fase 3 | 1-2 días | Fase 2 | Config YAML extendida |
| Fase 4 | 2-3 días | Fase 1-3 | Input hooks integrados |
| Fase 5 | 2-3 días | Fase 4 | Output hooks integrados |
| Fase 6 | 3-4 días | Fase 1-5 | Tests + docs + ejemplos |

**TOTAL: 12-18 días laborables (2.5-3.5 semanas)**

---

## Siguiente Paso

Para comenzar la implementación, ejecutar:

```bash
# 1. Crear branch de desarrollo
git checkout -b feature/lua-filters

# 2. Instalar dependencias (Fase 1)
go get github.com/yuin/gopher-lua
go get github.com/vadv/gopher-lua-libs/strings
# ... (ver sección Dependencias Técnicas)

# 3. Crear estructura de directorios
mkdir -p internal/luafilters
mkdir -p examples/filters

# 4. Consultar detalles de Fase 1
# Ver fact: fase1_setup_lua en remembrances
```

---

## Referencias

- **Facts de Remembrances**:
  - `fase1_setup_lua`: Detalles de configuración inicial
  - `fase2_sistema_hooks`: Interfaces y tipos del sistema
  - `fase3_config_yaml`: Extensión de configuración
  - `fase4_hooks_input`: Integración de filtros de entrada
  - `fase5_hooks_output`: Integración de filtros de salida
  - `fase6_testing_docs`: Testing y documentación

- **Proyecto de Referencia**: https://github.com/sevir/essh
- **Ubicación del Plan**: remembrances KB → `plans/lua-filters-implementation-plan.md`

---

**Documento preparado para**: Sevir
**Proyecto**: Panorganon
**Fecha**: 2026-01-28