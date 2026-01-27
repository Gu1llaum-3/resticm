// Package logging provides structured logging for resticm
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level represents log level
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is a structured logger
type Logger struct {
	mu       sync.Mutex
	level    Level
	outputs  []io.Writer
	prefix   string
	jsonMode bool
}

// Config represents logging configuration
type Config struct {
	File      string
	MaxSizeMB int
	MaxFiles  int
	Level     string
	Console   bool
	JSON      bool
}

var defaultLogger = NewLogger(INFO)

// NewLogger creates a new logger
func NewLogger(level Level) *Logger {
	return &Logger{
		level:   level,
		outputs: []io.Writer{os.Stdout},
	}
}

// Configure sets up the logger from config
func Configure(cfg Config) (*Logger, error) {
	level := parseLevel(cfg.Level)
	logger := NewLogger(level)
	logger.jsonMode = cfg.JSON

	var outputs []io.Writer

	if cfg.Console {
		outputs = append(outputs, os.Stdout)
	}

	if cfg.File != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		outputs = append(outputs, file)
	}

	if len(outputs) == 0 {
		outputs = append(outputs, os.Stdout)
	}

	logger.outputs = outputs
	return logger, nil
}

func parseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return DEBUG
	case "info", "INFO":
		return INFO
	case "warn", "WARN", "warning", "WARNING":
		return WARN
	case "error", "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetPrefix sets the log prefix
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// log writes a log entry
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)

	var line string
	if l.jsonMode {
		line = fmt.Sprintf(`{"time":"%s","level":"%s","msg":"%s"}`, now, level, msg)
	} else {
		if l.prefix != "" {
			line = fmt.Sprintf("[%s] [%s] [%s] %s", now, level, l.prefix, msg)
		} else {
			line = fmt.Sprintf("[%s] [%s] %s", now, level, msg)
		}
	}

	for _, out := range l.outputs {
		_, _ = fmt.Fprintln(out, line)
	}
}

// Debug logs at debug level
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs at info level
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs at error level
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal logs at error level and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
	os.Exit(1)
}

// WithPrefix returns a new logger with a prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	newLogger := &Logger{
		level:    l.level,
		outputs:  l.outputs,
		prefix:   prefix,
		jsonMode: l.jsonMode,
	}
	return newLogger
}

// Package-level functions using default logger

// Debug logs at debug level
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs at info level
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs at warn level
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs at error level
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs at error level and exits
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// SetDefault sets the default logger
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Init initializes logging from standard library
func Init(prefix string) {
	log.SetPrefix(prefix + " ")
	log.SetFlags(log.Ldate | log.Ltime)
}
