package main

import (
	"testing"
)

func TestIsValidLogLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		level string
		want  bool
	}{
		{"TRACE", true},
		{"DEBUG", true},
		{"INFO", true},
		{"WARN", true},
		{"ERROR", true},
		{"trace", true},
		{"info", true},
		{"Info", true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range cases {
		if got := isValidLogLevel(tt.level); got != tt.want {
			t.Errorf("isValidLogLevel(%q) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestLogLevel(t *testing.T) {
	cases := []struct {
		name    string
		envVal  string
		want    string
	}{
		{"empty env returns empty", "", ""},
		{"valid level uppercased", "info", "INFO"},
		{"already uppercase", "DEBUG", "DEBUG"},
		{"mixed case", "Warn", "WARN"},
		{"invalid falls back to TRACE", "bogus", "TRACE"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG", tt.envVal)
			got := LogLevel()
			if got != tt.want {
				t.Errorf("LogLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLogOutput_Discard(t *testing.T) {
	t.Setenv("LOG", "")
	out, err := logOutput()
	if err != nil {
		t.Fatalf("logOutput: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil writer")
	}
}

func TestLogOutput_WithLevel(t *testing.T) {
	t.Setenv("LOG", "DEBUG")
	// Unset LOG_PATH so it writes to stderr
	t.Setenv("LOG_PATH", "")
	out, err := logOutput()
	if err != nil {
		t.Fatalf("logOutput: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil writer")
	}
}

func TestLogOutput_WithLogPath(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"
	t.Setenv("LOG", "INFO")
	t.Setenv("LOG_PATH", tmpFile)
	out, err := logOutput()
	if err != nil {
		t.Fatalf("logOutput: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil writer")
	}
}

func TestLogOutput_InvalidLogPath(t *testing.T) {
	t.Setenv("LOG", "INFO")
	t.Setenv("LOG_PATH", "/nonexistent/dir/test.log")
	_, err := logOutput()
	if err == nil {
		t.Fatal("expected error for invalid log path")
	}
}
