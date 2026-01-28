package luafilters

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewFilterManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("disabled filters", func(t *testing.T) {
		config := FilterManagerConfig{
			Enabled: false,
			Logger:  logger,
		}

		fm, err := NewFilterManager(config)
		if err != nil {
			t.Fatalf("NewFilterManager() error = %v", err)
		}
		defer fm.Close()

		if fm.IsEnabled() {
			t.Error("FilterManager should be disabled")
		}

		if fm.IsScriptLoaded() {
			t.Error("Script should not be loaded when disabled")
		}
	})

	t.Run("enabled with valid script", func(t *testing.T) {
		config := FilterManagerConfig{
			Enabled:    true,
			ScriptPath: "testdata/test-filters.lua",
			Timeout:    5 * time.Second,
			StrictMode: false,
			Logger:     logger,
		}

		fm, err := NewFilterManager(config)
		if err != nil {
			t.Fatalf("NewFilterManager() error = %v", err)
		}
		defer fm.Close()

		if !fm.IsEnabled() {
			t.Error("FilterManager should be enabled")
		}

		if !fm.IsScriptLoaded() {
			t.Error("Script should be loaded")
		}
	})

	t.Run("enabled with invalid script path in non-strict mode", func(t *testing.T) {
		config := FilterManagerConfig{
			Enabled:    true,
			ScriptPath: "testdata/nonexistent.lua",
			Timeout:    5 * time.Second,
			StrictMode: false,
			Logger:     logger,
		}

		fm, err := NewFilterManager(config)
		if err != nil {
			t.Fatalf("NewFilterManager() should not fail in non-strict mode with invalid script: %v", err)
		}
		defer fm.Close()

		if !fm.IsEnabled() {
			t.Error("FilterManager should still be enabled even if script fails to load in non-strict mode")
		}

		if fm.IsScriptLoaded() {
			t.Error("Script should not be loaded when file doesn't exist")
		}
	})

	t.Run("strict mode with invalid script", func(t *testing.T) {
		config := FilterManagerConfig{
			Enabled:    true,
			ScriptPath: "testdata/nonexistent.lua",
			Timeout:    5 * time.Second,
			StrictMode: true,
			Logger:     logger,
		}

		_, err := NewFilterManager(config)
		if err == nil {
			t.Error("NewFilterManager() should fail in strict mode with invalid script")
		}
	})
}

func TestApplyInputFilter(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled:    true,
		ScriptPath: "testdata/test-filters.lua",
		Timeout:    5 * time.Second,
		StrictMode: false,
		Logger:     logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}
	defer fm.Close()

	ctx := context.Background()

	t.Run("filter exists and modifies parameters", func(t *testing.T) {
		params := map[string]interface{}{
			"query": "test query",
			"limit": 10,
		}

		hookCtx := NewInputContext("test-server", "search", params, "req-123")

		result, err := fm.ApplyInputFilter(ctx, hookCtx)
		if err != nil {
			t.Errorf("ApplyInputFilter() error = %v", err)
		}

		if !result.Modified {
			t.Error("ApplyInputFilter() should mark result as modified")
		}

		if filtered, ok := result.Data["filtered"].(bool); !ok || !filtered {
			t.Error("ApplyInputFilter() should add 'filtered' field")
		}

		if _, ok := result.Data["timestamp"].(int64); !ok {
			t.Error("ApplyInputFilter() should add 'timestamp' field")
		}
	})

	t.Run("filter does not exist - fallback to global", func(t *testing.T) {
		params := map[string]interface{}{
			"query": "test query",
		}

		hookCtx := NewInputContext("nonexistent-server", "search", params, "req-456")

		result, err := fm.ApplyInputFilter(ctx, hookCtx)
		if err != nil {
			t.Errorf("ApplyInputFilter() error = %v", err)
		}

		// Should use global-input filter
		if globalFilter, ok := result.Data["global_filter"].(bool); !ok || !globalFilter {
			t.Error("ApplyInputFilter() should apply global filter when specific filter doesn't exist")
		}
	})

	t.Run("filter error in non-strict mode", func(t *testing.T) {
		params := map[string]interface{}{
			"query": "test query",
		}

		hookCtx := NewInputContext("error-server", "search", params, "req-789")

		result, err := fm.ApplyInputFilter(ctx, hookCtx)

		// In non-strict mode, should return original parameters
		if err != nil {
			t.Error("ApplyInputFilter() should not return error in non-strict mode")
		}

		if result.Modified {
			t.Error("ApplyInputFilter() should not modify parameters when filter errors in non-strict mode")
		}
	})
}

func TestApplyOutputFilter(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled:    true,
		ScriptPath: "testdata/test-filters.lua",
		Timeout:    5 * time.Second,
		StrictMode: false,
		Logger:     logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}
	defer fm.Close()

	ctx := context.Background()

	t.Run("filter exists and modifies result", func(t *testing.T) {
		resultData := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "test result",
				},
			},
		}

		hookCtx := NewOutputContext("test-server", "search", resultData, "req-123", 100*time.Millisecond)

		result, err := fm.ApplyOutputFilter(ctx, hookCtx)
		if err != nil {
			t.Errorf("ApplyOutputFilter() error = %v", err)
		}

		if !result.Modified {
			t.Error("ApplyOutputFilter() should mark result as modified")
		}

		if filtered, ok := result.Data["filtered_output"].(bool); !ok || !filtered {
			t.Error("ApplyOutputFilter() should add 'filtered_output' field")
		}

		if duration, ok := result.Data["duration"].(int64); !ok || duration != 100 {
			t.Errorf("ApplyOutputFilter() duration = %v, want 100", result.Data["duration"])
		}
	})

	t.Run("filter does not exist - fallback to global", func(t *testing.T) {
		resultData := map[string]interface{}{
			"content": []interface{}{},
		}

		hookCtx := NewOutputContext("nonexistent-server", "search", resultData, "req-456", 50*time.Millisecond)

		result, err := fm.ApplyOutputFilter(ctx, hookCtx)
		if err != nil {
			t.Errorf("ApplyOutputFilter() error = %v", err)
		}

		// Should use global-output filter
		if globalFilter, ok := result.Data["global_filter"].(bool); !ok || !globalFilter {
			t.Error("ApplyOutputFilter() should apply global filter when specific filter doesn't exist")
		}
	})
}

func TestFilterTimeout(t *testing.T) {
	// Skip this test in normal runs as it can cause race conditions with goroutines
	// Run separately with: go test -run TestFilterTimeout
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled:    true,
		ScriptPath: "testdata/test-filters.lua",
		Timeout:    100 * time.Millisecond, // Very short timeout
		StrictMode: false,
		Logger:     logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}
	defer fm.Close()

	ctx := context.Background()

	t.Run("filter timeout in non-strict mode", func(t *testing.T) {
		params := map[string]interface{}{
			"query": "test query",
		}

		hookCtx := NewInputContext("timeout-server", "search", params, "req-timeout")

		result, err := fm.ApplyInputFilter(ctx, hookCtx)

		// In non-strict mode, should return original parameters on timeout
		if err != nil {
			t.Error("ApplyInputFilter() should not return error in non-strict mode on timeout")
		}

		if result.Modified {
			t.Error("ApplyInputFilter() should not modify parameters when filter times out in non-strict mode")
		}

		// Wait for background goroutine to finish
		time.Sleep(1 * time.Second)
	})
}

func TestReloadScript(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled:    true,
		ScriptPath: "testdata/test-filters.lua",
		Timeout:    5 * time.Second,
		StrictMode: false,
		Logger:     logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}
	defer fm.Close()

	if !fm.IsScriptLoaded() {
		t.Error("Script should be loaded initially")
	}

	// Give any pending goroutines time to complete
	time.Sleep(200 * time.Millisecond)

	// Reload the script
	err = fm.ReloadScript()
	if err != nil {
		t.Errorf("ReloadScript() error = %v", err)
	}

	if !fm.IsScriptLoaded() {
		t.Error("Script should still be loaded after reload")
	}

	// Verify the reloaded script works by executing a simple filter
	ctx := context.Background()
	params := map[string]interface{}{
		"test": "value",
	}
	hookCtx := NewInputContext("test-server", "search", params, "req-reload")

	result, err := fm.ApplyInputFilter(ctx, hookCtx)
	if err != nil {
		t.Errorf("ApplyInputFilter() after reload error = %v", err)
	}

	if !result.Modified {
		t.Error("Filter should work after reload")
	}
}

func TestBuildFunctionName(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled: false,
		Logger:  logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}
	defer fm.Close()

	tests := []struct {
		serverName string
		filterType FilterType
		expected   string
	}{
		{"test-server", FilterTypeInput, "test-server-input"},
		{"test-server", FilterTypeOutput, "test-server-output"},
		{"test_server", FilterTypeInput, "test-server-input"},
		{"test.server", FilterTypeInput, "test-server-input"},
		{"complex_server.name", FilterTypeOutput, "complex-server-name-output"},
	}

	for _, tt := range tests {
		t.Run(tt.serverName, func(t *testing.T) {
			result := fm.buildFunctionName(tt.serverName, tt.filterType)
			if result != tt.expected {
				t.Errorf("buildFunctionName(%s, %s) = %s, want %s",
					tt.serverName, tt.filterType, result, tt.expected)
			}
		})
	}
}

func TestFilterManagerClose(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled:    true,
		ScriptPath: "testdata/test-filters.lua",
		Timeout:    5 * time.Second,
		Logger:     logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}

	// Close should not panic
	fm.Close()

	// Double close should not panic
	fm.Close()
}

func TestDisabledFilterManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := FilterManagerConfig{
		Enabled: false,
		Logger:  logger,
	}

	fm, err := NewFilterManager(config)
	if err != nil {
		t.Fatalf("NewFilterManager() error = %v", err)
	}
	defer fm.Close()

	ctx := context.Background()

	params := map[string]interface{}{
		"query": "test query",
	}

	hookCtx := NewInputContext("test-server", "search", params, "req-123")

	result, err := fm.ApplyInputFilter(ctx, hookCtx)
	if err != nil {
		t.Errorf("ApplyInputFilter() with disabled manager should not error: %v", err)
	}

	if result.Modified {
		t.Error("ApplyInputFilter() with disabled manager should not modify parameters")
	}

	// Should return original parameters
	if result.Data["query"] != "test query" {
		t.Error("ApplyInputFilter() with disabled manager should return original parameters")
	}
}
