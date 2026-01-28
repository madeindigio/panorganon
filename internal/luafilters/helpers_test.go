package luafilters

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestGoToLua(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	tests := []struct {
		name     string
		input    interface{}
		expected lua.LValueType
	}{
		{"nil", nil, lua.LTNil},
		{"bool true", true, lua.LTBool},
		{"bool false", false, lua.LTBool},
		{"string", "test", lua.LTString},
		{"int", 42, lua.LTNumber},
		{"int64", int64(42), lua.LTNumber},
		{"float32", float32(3.14), lua.LTNumber},
		{"float64", 3.14, lua.LTNumber},
		{"map", map[string]interface{}{"key": "value"}, lua.LTTable},
		{"slice", []interface{}{1, 2, 3}, lua.LTTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GoToLua(L, tt.input)
			if result.Type() != tt.expected {
				t.Errorf("GoToLua(%v) type = %v, want %v", tt.input, result.Type(), tt.expected)
			}
		})
	}
}

func TestLuaToGo(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	tests := []struct {
		name     string
		setup    func() lua.LValue
		expected interface{}
	}{
		{
			name:     "nil",
			setup:    func() lua.LValue { return lua.LNil },
			expected: nil,
		},
		{
			name:     "bool true",
			setup:    func() lua.LValue { return lua.LBool(true) },
			expected: true,
		},
		{
			name:     "bool false",
			setup:    func() lua.LValue { return lua.LBool(false) },
			expected: false,
		},
		{
			name:     "string",
			setup:    func() lua.LValue { return lua.LString("test") },
			expected: "test",
		},
		{
			name:     "integer number",
			setup:    func() lua.LValue { return lua.LNumber(42) },
			expected: int64(42),
		},
		{
			name:     "float number",
			setup:    func() lua.LValue { return lua.LNumber(3.14) },
			expected: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lval := tt.setup()
			result := LuaToGo(lval)

			// Type assertion based on expected type
			switch exp := tt.expected.(type) {
			case nil:
				if result != nil {
					t.Errorf("LuaToGo() = %v, want nil", result)
				}
			case bool:
				if res, ok := result.(bool); !ok || res != exp {
					t.Errorf("LuaToGo() = %v, want %v", result, exp)
				}
			case string:
				if res, ok := result.(string); !ok || res != exp {
					t.Errorf("LuaToGo() = %v, want %v", result, exp)
				}
			case int64:
				if res, ok := result.(int64); !ok || res != exp {
					t.Errorf("LuaToGo() = %v, want %v", result, exp)
				}
			case float64:
				if res, ok := result.(float64); !ok || res != exp {
					t.Errorf("LuaToGo() = %v, want %v", result, exp)
				}
			}
		})
	}
}

func TestMapToLuaTable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	input := map[string]interface{}{
		"name":   "test",
		"count":  42,
		"active": true,
	}

	table := MapToLuaTable(L, input)

	if table.Type() != lua.LTTable {
		t.Errorf("MapToLuaTable() type = %v, want LTTable", table.Type())
	}

	// Check field values
	if name := table.RawGetString("name"); name.String() != "test" {
		t.Errorf("table.name = %v, want 'test'", name)
	}

	if count := table.RawGetString("count"); count.Type() != lua.LTNumber {
		t.Errorf("table.count type = %v, want LTNumber", count.Type())
	}

	if active := table.RawGetString("active"); active.Type() != lua.LTBool {
		t.Errorf("table.active type = %v, want LTBool", active.Type())
	}
}

func TestSliceToLuaTable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	input := []interface{}{1, 2, 3, "four", true}

	table := SliceToLuaTable(L, input)

	if table.Type() != lua.LTTable {
		t.Errorf("SliceToLuaTable() type = %v, want LTTable", table.Type())
	}

	// Check array length (Lua arrays are 1-indexed)
	if table.Len() != len(input) {
		t.Errorf("table.Len() = %d, want %d", table.Len(), len(input))
	}

	// Check first element
	if first := table.RawGetInt(1); first.Type() != lua.LTNumber {
		t.Errorf("table[1] type = %v, want LTNumber", first.Type())
	}

	// Check last element
	if last := table.RawGetInt(5); last.Type() != lua.LTBool {
		t.Errorf("table[5] type = %v, want LTBool", last.Type())
	}
}

func TestLuaTableToGo(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	t.Run("array", func(t *testing.T) {
		// Create Lua array
		table := L.NewTable()
		table.RawSetInt(1, lua.LNumber(1))
		table.RawSetInt(2, lua.LNumber(2))
		table.RawSetInt(3, lua.LNumber(3))

		result := LuaTableToGo(table)

		slice, ok := result.([]interface{})
		if !ok {
			t.Errorf("LuaTableToGo(array) type = %T, want []interface{}", result)
			return
		}

		if len(slice) != 3 {
			t.Errorf("len(slice) = %d, want 3", len(slice))
		}
	})

	t.Run("map", func(t *testing.T) {
		// Create Lua table with string keys
		table := L.NewTable()
		table.RawSetString("name", lua.LString("test"))
		table.RawSetString("count", lua.LNumber(42))

		result := LuaTableToGo(table)

		m, ok := result.(map[string]interface{})
		if !ok {
			t.Errorf("LuaTableToGo(map) type = %T, want map[string]interface{}", result)
			return
		}

		if len(m) != 2 {
			t.Errorf("len(map) = %d, want 2", len(m))
		}

		if name, ok := m["name"].(string); !ok || name != "test" {
			t.Errorf("map['name'] = %v, want 'test'", m["name"])
		}
	})
}

func TestIsLuaArray(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	t.Run("is array", func(t *testing.T) {
		table := L.NewTable()
		table.RawSetInt(1, lua.LNumber(1))
		table.RawSetInt(2, lua.LNumber(2))
		table.RawSetInt(3, lua.LNumber(3))

		if !isLuaArray(table) {
			t.Error("isLuaArray() = false, want true for sequential array")
		}
	})

	t.Run("is not array", func(t *testing.T) {
		table := L.NewTable()
		table.RawSetString("key", lua.LString("value"))

		if isLuaArray(table) {
			t.Error("isLuaArray() = true, want false for map")
		}
	})

	t.Run("empty table", func(t *testing.T) {
		table := L.NewTable()

		if !isLuaArray(table) {
			t.Error("isLuaArray() = false, want true for empty table")
		}
	})
}

func TestGetSetTableField(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	table := L.NewTable()

	// Test SetTableField
	SetTableField(L, table, "name", "test")
	SetTableField(L, table, "count", 42)

	// Test GetTableField
	name := GetTableField(L, table, "name")
	if name.Type() != lua.LTString {
		t.Errorf("GetTableField('name') type = %v, want LTString", name.Type())
	}

	count := GetTableField(L, table, "count")
	if count.Type() != lua.LTNumber {
		t.Errorf("GetTableField('count') type = %v, want LTNumber", count.Type())
	}
}

func TestGetTableFieldAsString(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	table := L.NewTable()
	table.RawSetString("name", lua.LString("test"))
	table.RawSetString("count", lua.LNumber(42))

	t.Run("string field", func(t *testing.T) {
		value, ok := GetTableFieldAsString(L, table, "name")
		if !ok {
			t.Error("GetTableFieldAsString('name') ok = false, want true")
		}
		if value != "test" {
			t.Errorf("GetTableFieldAsString('name') = %v, want 'test'", value)
		}
	})

	t.Run("non-string field", func(t *testing.T) {
		_, ok := GetTableFieldAsString(L, table, "count")
		if ok {
			t.Error("GetTableFieldAsString('count') ok = true, want false")
		}
	})

	t.Run("missing field", func(t *testing.T) {
		_, ok := GetTableFieldAsString(L, table, "missing")
		if ok {
			t.Error("GetTableFieldAsString('missing') ok = true, want false")
		}
	})
}

func TestRoundTripConversion(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Test Go -> Lua -> Go round trip
	input := map[string]interface{}{
		"name": "test",
		"nested": map[string]interface{}{
			"count": 42,
			"items": []interface{}{1, 2, 3},
		},
	}

	// Convert to Lua
	luaTable := MapToLuaTable(L, input)

	// Convert back to Go
	output := LuaTableToGo(luaTable)

	outputMap, ok := output.(map[string]interface{})
	if !ok {
		t.Fatalf("Round trip conversion failed: output is not a map")
	}

	if name, ok := outputMap["name"].(string); !ok || name != "test" {
		t.Errorf("Round trip: name = %v, want 'test'", outputMap["name"])
	}

	if nested, ok := outputMap["nested"].(map[string]interface{}); !ok {
		t.Error("Round trip: nested is not a map")
	} else {
		if count, ok := nested["count"].(int64); !ok || count != 42 {
			t.Errorf("Round trip: nested.count = %v, want 42", nested["count"])
		}
	}
}
