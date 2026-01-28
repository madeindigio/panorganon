package luafilters

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// GoToLua converts a Go value to a Lua value
// Supports: nil, bool, string, int, int64, float32, float64, map[string]interface{}, []interface{}
func GoToLua(L *lua.LState, val interface{}) lua.LValue {
	switch v := val.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(v)
	case string:
		return lua.LString(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case float32:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case map[string]interface{}:
		return MapToLuaTable(L, v)
	case []interface{}:
		return SliceToLuaTable(L, v)
	default:
		// For unsupported types, return nil and log a warning
		return lua.LNil
	}
}

// LuaToGo converts a Lua value to a Go value
// Supports: LNil, LBool, LString, LNumber, LTable
func LuaToGo(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		// Check if it's an integer or float
		if float64(v) == float64(int64(v)) {
			return int64(v)
		}
		return float64(v)
	case *lua.LTable:
		return LuaTableToGo(v)
	default:
		return nil
	}
}

// MapToLuaTable converts a Go map[string]interface{} to a Lua table
func MapToLuaTable(L *lua.LState, m map[string]interface{}) *lua.LTable {
	table := L.NewTable()
	for k, v := range m {
		table.RawSetString(k, GoToLua(L, v))
	}
	return table
}

// SliceToLuaTable converts a Go []interface{} to a Lua table (array)
// Lua arrays are 1-indexed
func SliceToLuaTable(L *lua.LState, s []interface{}) *lua.LTable {
	table := L.NewTable()
	for i, v := range s {
		table.RawSetInt(i+1, GoToLua(L, v)) // Lua arrays start at 1
	}
	return table
}

// LuaTableToGo converts a Lua table to a Go value
// Detects if the table is an array or a map and converts accordingly
func LuaTableToGo(lt *lua.LTable) interface{} {
	// Check if the table is an array (sequential integer keys starting from 1)
	if isLuaArray(lt) {
		return luaTableToSlice(lt)
	}
	// Otherwise, treat it as a map
	return luaTableToMap(lt)
}

// isLuaArray checks if a Lua table is an array (sequential integer keys)
func isLuaArray(lt *lua.LTable) bool {
	// Check if all keys are sequential integers starting from 1
	length := lt.Len()
	if length == 0 {
		// Empty table - check if it has any non-integer keys
		hasNonIntKeys := false
		lt.ForEach(func(k, v lua.LValue) {
			if _, ok := k.(lua.LNumber); !ok {
				hasNonIntKeys = true
			}
		})
		return !hasNonIntKeys
	}

	// If table has length, it's likely an array
	// Verify all keys from 1 to length exist
	for i := 1; i <= length; i++ {
		if lt.RawGetInt(i) == lua.LNil {
			return false
		}
	}

	// Check if there are any non-integer keys
	hasNonIntKeys := false
	lt.ForEach(func(k, v lua.LValue) {
		if num, ok := k.(lua.LNumber); ok {
			idx := int(num)
			if idx < 1 || idx > length {
				hasNonIntKeys = true
			}
		} else {
			hasNonIntKeys = true
		}
	})

	return !hasNonIntKeys
}

// luaTableToSlice converts a Lua array to a Go slice
func luaTableToSlice(lt *lua.LTable) []interface{} {
	length := lt.Len()
	result := make([]interface{}, length)
	for i := 1; i <= length; i++ {
		result[i-1] = LuaToGo(lt.RawGetInt(i))
	}
	return result
}

// luaTableToMap converts a Lua table to a Go map
func luaTableToMap(lt *lua.LTable) map[string]interface{} {
	result := make(map[string]interface{})
	lt.ForEach(func(k, v lua.LValue) {
		key := ""
		switch kt := k.(type) {
		case lua.LString:
			key = string(kt)
		case lua.LNumber:
			key = fmt.Sprintf("%v", kt)
		default:
			key = k.String()
		}
		result[key] = LuaToGo(v)
	})
	return result
}

// GetTableField safely gets a field from a Lua table
func GetTableField(L *lua.LState, table *lua.LTable, key string) lua.LValue {
	return table.RawGetString(key)
}

// SetTableField safely sets a field in a Lua table
func SetTableField(L *lua.LState, table *lua.LTable, key string, value interface{}) {
	table.RawSetString(key, GoToLua(L, value))
}

// GetTableFieldAsString gets a field from a Lua table as a Go string
func GetTableFieldAsString(L *lua.LState, table *lua.LTable, key string) (string, bool) {
	val := table.RawGetString(key)
	if str, ok := val.(lua.LString); ok {
		return string(str), true
	}
	return "", false
}

// GetTableFieldAsMap gets a field from a Lua table as a Go map
func GetTableFieldAsMap(L *lua.LState, table *lua.LTable, key string) (map[string]interface{}, bool) {
	val := table.RawGetString(key)
	if tbl, ok := val.(*lua.LTable); ok {
		m := luaTableToMap(tbl)
		return m, true
	}
	return nil, false
}

// GetTableFieldAsSlice gets a field from a Lua table as a Go slice
func GetTableFieldAsSlice(L *lua.LState, table *lua.LTable, key string) ([]interface{}, bool) {
	val := table.RawGetString(key)
	if tbl, ok := val.(*lua.LTable); ok {
		if isLuaArray(tbl) {
			s := luaTableToSlice(tbl)
			return s, true
		}
	}
	return nil, false
}
