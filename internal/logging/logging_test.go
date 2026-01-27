package logging

import (
	"testing"
)

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     Level
		logLevel  Level
		shouldLog bool
	}{
		{"debug at debug level", DEBUG, DEBUG, true},
		{"info at debug level", INFO, DEBUG, true},
		{"warn at debug level", WARN, DEBUG, true},
		{"error at debug level", ERROR, DEBUG, true},
		{"debug at info level", DEBUG, INFO, false},
		{"info at info level", INFO, INFO, true},
		{"warn at info level", WARN, INFO, true},
		{"error at info level", ERROR, INFO, true},
		{"debug at warn level", DEBUG, WARN, false},
		{"info at warn level", INFO, WARN, false},
		{"warn at warn level", WARN, WARN, true},
		{"error at warn level", ERROR, WARN, true},
		{"debug at error level", DEBUG, ERROR, false},
		{"info at error level", INFO, ERROR, false},
		{"warn at error level", WARN, ERROR, false},
		{"error at error level", ERROR, ERROR, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldLog := tt.level >= tt.logLevel
			if shouldLog != tt.shouldLog {
				t.Errorf("level %v >= %v = %v, want %v", tt.level, tt.logLevel, shouldLog, tt.shouldLog)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"WARN", WARN},
		{"warning", WARN},
		{"WARNING", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"unknown", INFO}, // default
		{"", INFO},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoggerWithPrefix(t *testing.T) {
	logger := &Logger{
		level:  INFO,
		prefix: "test",
	}

	newLogger := logger.WithPrefix("submodule")
	if newLogger.prefix != "submodule" {
		t.Errorf("WithPrefix() prefix = %q, want %q", newLogger.prefix, "submodule")
	}

	// Original should be unchanged
	if logger.prefix != "test" {
		t.Errorf("Original logger prefix = %q, want %q", logger.prefix, "test")
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger(WARN)

	if logger.level != WARN {
		t.Errorf("level = %v, want %v", logger.level, WARN)
	}

	if len(logger.outputs) != 1 {
		t.Errorf("len(outputs) = %d, want 1", len(logger.outputs))
	}

	if logger.jsonMode {
		t.Error("jsonMode should be false by default")
	}
}

func TestConfigureWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := tmpDir + "/test.log"

	cfg := Config{
		File:    logFile,
		Level:   "debug",
		Console: false,
		JSON:    true,
	}

	logger, err := Configure(cfg)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	if logger.level != DEBUG {
		t.Errorf("level = %v, want %v", logger.level, DEBUG)
	}

	if !logger.jsonMode {
		t.Error("jsonMode should be true")
	}

	if len(logger.outputs) != 1 {
		t.Errorf("len(outputs) = %d, want 1 (file only)", len(logger.outputs))
	}
}

func TestConfigureWithConsole(t *testing.T) {
	cfg := Config{
		Level:   "info",
		Console: true,
		JSON:    false,
	}

	logger, err := Configure(cfg)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	if logger.level != INFO {
		t.Errorf("level = %v, want %v", logger.level, INFO)
	}

	if logger.jsonMode {
		t.Error("jsonMode should be false")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Verify default logger exists
	if defaultLogger == nil {
		t.Error("defaultLogger should not be nil")
	}

	// Test SetDefault
	newLogger := NewLogger(ERROR)
	SetDefault(newLogger)

	if defaultLogger != newLogger {
		t.Error("SetDefault did not update defaultLogger")
	}

	// Restore default
	SetDefault(NewLogger(INFO))
}

func TestInit(t *testing.T) {
	// This just sets up standard library logger
	Init("resticm")
	// No error means success
}

func TestLogMethods(t *testing.T) {
	// Test that package-level functions don't panic
	// We can't easily capture output, but we can verify no panic

	testLogger := NewLogger(ERROR) // Set to ERROR so nothing is actually logged
	SetDefault(testLogger)

	// These should not panic
	Debug("test debug %s", "message")
	Info("test info %s", "message")
	Warn("test warn %s", "message")
	Error("test error %s", "message")
}

func TestLevelComparison(t *testing.T) {
	if DEBUG >= INFO {
		t.Error("DEBUG should be less than INFO")
	}
	if INFO >= WARN {
		t.Error("INFO should be less than WARN")
	}
	if WARN >= ERROR {
		t.Error("WARN should be less than ERROR")
	}
}

func TestLoggerMutex(t *testing.T) {
	// Verify logger has mutex for thread safety
	logger := NewLogger(INFO)

	// Concurrent access should not cause race conditions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			logger.SetLevel(Level(n % 4))
			logger.SetPrefix(string(rune('a' + n)))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestOutputsSlice(t *testing.T) {
	logger := NewLogger(INFO)

	// Verify outputs is not nil
	if logger.outputs == nil {
		t.Error("outputs should not be nil")
	}

	// Should have stdout by default
	if len(logger.outputs) == 0 {
		t.Error("outputs should have at least one writer")
	}
}

func TestLoggerState(t *testing.T) {
	logger := NewLogger(INFO)

	// Verify state was set
	logger.SetLevel(DEBUG)
	logger.SetPrefix("test")

	if logger.level != DEBUG {
		t.Errorf("level = %v, want %v", logger.level, DEBUG)
	}

	if logger.prefix != "test" {
		t.Errorf("prefix = %q, want %q", logger.prefix, "test")
	}
}
