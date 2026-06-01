package logger

import (
	"log/slog"
	"os"
)

var L = &Logger{Slog: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))}

type Logger struct {
	Slog *slog.Logger
}

func Init(level slog.Level, env string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if env == "dev" || env == "local" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	L = &Logger{Slog: slog.New(handler)}
}

func (l *Logger) Debug(msg string, keyvals ...any) { l.Slog.Debug(msg, keyvals...) }
func (l *Logger) Info(msg string, keyvals ...any)  { l.Slog.Info(msg, keyvals...) }
func (l *Logger) Warn(msg string, keyvals ...any)  { l.Slog.Warn(msg, keyvals...) }
func (l *Logger) Error(msg string, keyvals ...any) { l.Slog.Error(msg, keyvals...) }
func (l *Logger) Fatal(msg string, keyvals ...any) {
	l.Slog.Error(msg, keyvals...)
	os.Exit(1)
}
func (l *Logger) With(keyvals ...any) *Logger {
	return &Logger{Slog: l.Slog.With(keyvals...)}
}
