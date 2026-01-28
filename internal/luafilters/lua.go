package luafilters

import (
	"github.com/layeh/gopher-json"
	lua "github.com/yuin/gopher-lua"

	// Import gopher-lua-libs modules
	env "github.com/vadv/gopher-lua-libs/argparse"
	strings "github.com/vadv/gopher-lua-libs/strings"
	http "github.com/vadv/gopher-lua-libs/http"
	time "github.com/vadv/gopher-lua-libs/time"
	re "github.com/vadv/gopher-lua-libs/regexp"
	yaml "github.com/vadv/gopher-lua-libs/yaml"
	template "github.com/vadv/gopher-lua-libs/template"
)

// InitLuaState initializes a new Lua state with all required modules preloaded
// This function sets up a sandboxed Lua environment suitable for filter execution
func InitLuaState() (*lua.LState, error) {
	// Create new Lua state with default options
	L := lua.NewState(lua.Options{
		CallStackSize:       120,  // Maximum call stack size
		RegistrySize:        1024, // Initial registry size
		SkipOpenLibs:        false, // Load standard libraries
		IncludeGoStackTrace: true, // Include Go stack trace in errors
	})

	// Preload standard Lua libraries
	// Basic libraries are loaded by default when SkipOpenLibs is false

	// Load gopher-lua-libs modules
	// These provide extended functionality for string manipulation, HTTP, etc.

	// strings: Advanced string manipulation functions
	strings.Preload(L)

	// http: HTTP client functionality (with restrictions)
	http.Preload(L)

	// time: Time and date manipulation
	time.Preload(L)

	// re: Regular expression support using Go regexp
	re.Preload(L)

	// yaml: YAML parsing and generation
	yaml.Preload(L)

	// template: Text template processing
	template.Preload(L)

	// env: Environment variable access (argparse provides env functionality)
	env.Preload(L)

	// json: JSON encoding/decoding using gopher-json
	L.PreloadModule("json", json.Loader)

	// Note: sh module is intentionally NOT loaded for security reasons
	// Executing arbitrary shell commands from Lua filters would be a security risk

	return L, nil
}

// ResetLuaState clears the Lua state and reinitializes it
// Useful for hot-reloading filters or recovering from errors
func ResetLuaState(L *lua.LState) (*lua.LState, error) {
	if L != nil {
		L.Close()
	}
	return InitLuaState()
}

// CloseLuaState properly closes and cleans up a Lua state
func CloseLuaState(L *lua.LState) {
	if L != nil {
		L.Close()
	}
}
