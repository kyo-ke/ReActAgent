package logging

import (
	"io"
	"log"
	"os"
	"strings"
)

// Logger is the tiny logging interface used across this repo.
// It keeps dependencies small while allowing injection from main/tests.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// New returns a logger configured for this repo.
//
// Defaults:
//   - output: stderr
//   - level: info
//
// Env (optional):
//   - LOG_LEVEL: debug|info|warn|error
func New() Logger {
	return NewWithOptions(Options{})
}

type Options struct {
	Output io.Writer
	Level  Level
	Prefix string
}

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func NewWithOptions(opt Options) Logger {
	out := opt.Output
	if out == nil {
		out = os.Stderr
	}

	level := opt.Level
	if opt.Level == 0 {
		level = levelFromEnv("LOG_LEVEL", LevelInfo)
	}

	prefix := opt.Prefix
	if prefix == "" {
		prefix = ""
	}

	// stdlib logger is safe for concurrent use.
	base := log.New(out, prefix, log.Ldate|log.Ltime|log.Lmicroseconds)
	return &stdLogger{base: base, level: level}
}

func levelFromEnv(env string, fallback Level) Level {
	s := strings.ToLower(strings.TrimSpace(os.Getenv(env)))
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "":
		return fallback
	default:
		return fallback
	}
}

type stdLogger struct {
	base  *log.Logger
	level Level
}

func (l *stdLogger) Debugf(format string, args ...any) {
	if l.level > LevelDebug {
		return
	}
	l.base.Printf("DEBUG "+format, args...)
}

func (l *stdLogger) Infof(format string, args ...any) {
	if l.level > LevelInfo {
		return
	}
	l.base.Printf("INFO "+format, args...)
}

func (l *stdLogger) Warnf(format string, args ...any) {
	if l.level > LevelWarn {
		return
	}
	l.base.Printf("WARN "+format, args...)
}

func (l *stdLogger) Errorf(format string, args ...any) {
	if l.level > LevelError {
		return
	}
	l.base.Printf("ERROR "+format, args...)
}
