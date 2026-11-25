//go:build js && wasm

package web

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"syscall/js"
)

type consoleLogger struct {
	level *slog.LevelVar
	attrs []slog.Attr
}

func (l consoleLogger) Enabled(_ context.Context, level slog.Level) bool {
	return level >= l.level.Level()
}

func (l consoleLogger) Handle(_ context.Context, record slog.Record) error {
	var message strings.Builder
	fmt.Fprintf(&message, "[%s] %s: %s", logLevelName(record.Level), logTarget(record.PC), record.Message)
	for _, attr := range l.attrs {
		fmt.Fprintf(&message, " %s", attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		fmt.Fprintf(&message, " %s", attr)
		return true
	})
	console := js.Global().Get("console")
	method := "log"
	switch {
	case record.Level >= slog.LevelError:
		method = "error"
	case record.Level >= slog.LevelWarn:
		method = "warn"
	case record.Level >= slog.LevelInfo:
		method = "info"
	}
	_, _ = jsCall(console, method, message.String())
	return nil
}

func (l consoleLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	l.attrs = append(l.attrs[:len(l.attrs):len(l.attrs)], attrs...)
	return l
}

func (l consoleLogger) WithGroup(string) slog.Handler {
	return l
}

func logLevelName(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	case level >= slog.LevelDebug:
		return "DEBUG"
	}
	return "TRACE"
}

func logTarget(pc uintptr) string {
	if pc == 0 {
		return "gpui_web"
	}
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	name := frame.Function
	slash := strings.LastIndexByte(name, '/')
	if dot := strings.IndexByte(name[slash+1:], '.'); dot >= 0 {
		return name[:slash+1+dot]
	}
	return name
}

var (
	loggerOnce  sync.Once
	loggerLevel slog.LevelVar
)

// InitLogging routes log/slog records to the browser console at Info level and above.
func InitLogging() {
	loggerOnce.Do(func() {
		slog.SetDefault(slog.New(consoleLogger{level: &loggerLevel}))
	})
	loggerLevel.Set(slog.LevelInfo)
}
