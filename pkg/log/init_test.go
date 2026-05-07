package log

import (
	"testing"
)

func TestInit(t *testing.T) {
	// Test valid configurations
	tests := []struct {
		level  string
		format string
	}{
		{"info", "console"},
		{"debug", "json"},
		{"warn", "console"},
		{"error", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.level+"-"+tt.format, func(t *testing.T) {
			err := Init(tt.level, tt.format)
			if err != nil {
				t.Errorf("Expected no error for level=%s format=%s, got: %v", tt.level, tt.format, err)
			}
			if Logger == nil {
				t.Errorf("Expected global Logger to be set, but was nil")
			}
		})
	}

	// Test invalid log level
	err := Init("invalid_level", "console")
	if err == nil {
		t.Errorf("Expected error for invalid log level, got nil")
	}
}

func TestNoop(t *testing.T) {
	logger := Noop()
	if logger == nil {
		t.Errorf("Expected Noop() to return a non-nil logger")
	}

	// Ensure it doesn't panic when we log something
	logger.Info("this is a test")
}
