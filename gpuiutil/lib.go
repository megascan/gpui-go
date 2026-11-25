package gpuiutil

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"
)

// IsUTF8CharBoundary reports whether b can start a UTF-8 encoded character.
func IsUTF8CharBoundary(b byte) bool {
	return int8(b) >= -0x40
}

// Incrementable is the set of types PostInc accepts.
type Incrementable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// PostInc increments *value by one and returns the previous value.
func PostInc[T Incrementable](value *T) T {
	prev := *value
	*value += T(1)
	return prev
}

// CanonicalFloatBits returns the bits of value with the sign of zero normalised away.
func CanonicalFloatBits(value float32) uint32 {
	if value == 0.0 {
		return 0
	}
	return math.Float32bits(value)
}

var zedMeasurements = sync.OnceValue(func() bool {
	m, ok := os.LookupEnv("ZED_MEASUREMENTS")
	return ok && (m == "1" || m == "true")
})

// Measure runs f and, when ZED_MEASUREMENTS is set, prints how long it took.
func Measure[R any](label string, f func() R) R {
	if zedMeasurements() {
		start := time.Now()
		result := f()
		elapsed := time.Since(start)
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, elapsed)
		return result
	}
	return f()
}

// DebugPanic panics when debug assertions are on and logs an error with a stack trace otherwise.
func DebugPanic(format string, args ...any) {
	if DebugAssertions {
		panic(fmt.Sprintf(format, args...))
	}
	slog.Error(fmt.Sprintf("%s\n%s", fmt.Sprintf(format, args...), debug.Stack()))
}

// SomeOrDebugPanic passes its arguments through and panics in debug builds when ok is false.
func SomeOrDebugPanic[T any](value T, ok bool) (T, bool) {
	if DebugAssertions && !ok {
		panic("Unexpected None")
	}
	return value, ok
}

// Location is a call site captured for logging.
type Location struct {
	PC   uintptr
	File string
	Line int
}

// CallerLocation returns the location of the caller skip frames above the function calling it.
func CallerLocation(skip int) Location {
	var pcs [1]uintptr
	if runtime.Callers(skip+2, pcs[:]) == 0 {
		return Location{}
	}
	frame, _ := runtime.CallersFrames(pcs[:]).Next()
	return Location{PC: pcs[0], File: frame.File, Line: frame.Line}
}

// ResultExt carries a (value, error) pair and logs the error when it is consumed.
type ResultExt[T any] struct {
	Value T
	Err   error
}

// Result wraps a (value, error) pair for the ResultExt methods.
func Result[T any](value T, err error) ResultExt[T] {
	return ResultExt[T]{Value: value, Err: err}
}

// LogErr logs the error at error level and returns the value when there is none.
func (r ResultExt[T]) LogErr() (T, bool) {
	return r.logWithLevel(CallerLocation(1), slog.LevelError)
}

// LogErrWithBacktrace is LogErr with the error formatted by %+v.
func (r ResultExt[T]) LogErrWithBacktrace() (T, bool) {
	if r.Err == nil {
		return r.Value, true
	}
	logErrorWithCaller(CallerLocation(1), fmt.Sprintf("%+v", r.Err), slog.LevelError)
	var zero T
	return zero, false
}

// DebugAssertOk asserts that the result is not an error in development and tests.
func (r ResultExt[T]) DebugAssertOk(reason string) ResultExt[T] {
	if r.Err != nil {
		DebugPanic("%s - %v", reason, r.Err)
	}
	return r
}

// WarnOnErr logs the error at warn level and returns the value when there is none.
func (r ResultExt[T]) WarnOnErr() (T, bool) {
	return r.logWithLevel(CallerLocation(1), slog.LevelWarn)
}

// LogWithLevel logs the error at level and returns the value when there is none.
func (r ResultExt[T]) LogWithLevel(level slog.Level) (T, bool) {
	return r.logWithLevel(CallerLocation(1), level)
}

// Anyhow returns the pair unchanged.
func (r ResultExt[T]) Anyhow() (T, error) {
	return r.Value, r.Err
}

func (r ResultExt[T]) logWithLevel(loc Location, level slog.Level) (T, bool) {
	if r.Err == nil {
		return r.Value, true
	}
	logErrorWithCaller(loc, r.Err.Error(), level)
	var zero T
	return zero, false
}

func logErrorWithCaller(caller Location, message string, level slog.Level) {
	file := strings.ReplaceAll(caller.File, "\\", "/")
	var target, modulePath string
	if caller.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{caller.PC}).Next()
		pkg := packagePath(frame.Function)
		if pkg != "" {
			stem := strings.TrimSuffix(path.Base(file), ".go")
			target = path.Base(pkg) + "::" + stem
			modulePath = pkg + "/" + path.Base(file)
		}
	}
	handler := slog.Default().Handler()
	ctx := context.Background()
	if !handler.Enabled(ctx, level) {
		return
	}
	record := slog.NewRecord(time.Now(), level, message, caller.PC)
	record.AddAttrs(
		slog.String("target", target),
		slog.String("module_path", modulePath),
		slog.String("file", caller.File),
		slog.Int("line", caller.Line),
	)
	_ = handler.Handle(ctx, record)
}

func packagePath(function string) string {
	slash := strings.LastIndexByte(function, '/')
	dot := strings.IndexByte(function[slash+1:], '.')
	if dot < 0 {
		return ""
	}
	return function[:slash+1+dot]
}

// LogErr logs err at error level with the caller's location.
func LogErr(err error) {
	logErrorWithCaller(CallerLocation(1), err.Error(), slog.LevelError)
}

// TryFutureExt is a fallible asynchronous body; its methods wrap it so the error is logged or unwrapped.
type TryFutureExt[T any] func() (T, error)

// TryFutureExtBacktrace is TryFutureExt whose wrappers format the error with %+v.
type TryFutureExtBacktrace[T any] func() (T, error)

// LogErr wraps f so its error is logged at error level.
func (f TryFutureExt[T]) LogErr() LogErrorFuture[T] {
	return LogErrorFuture[T]{f: f, level: slog.LevelError, location: CallerLocation(1)}
}

// LogTrackedErr wraps f so its error is logged at error level with location.
func (f TryFutureExt[T]) LogTrackedErr(location Location) LogErrorFuture[T] {
	return LogErrorFuture[T]{f: f, level: slog.LevelError, location: location}
}

// WarnOnErr wraps f so its error is logged at warn level.
func (f TryFutureExt[T]) WarnOnErr() LogErrorFuture[T] {
	return LogErrorFuture[T]{f: f, level: slog.LevelWarn, location: CallerLocation(1)}
}

// Unwrap wraps f so its error panics.
func (f TryFutureExt[T]) Unwrap() UnwrapFuture[T] {
	return UnwrapFuture[T]{f: f}
}

// LogErrWithBacktrace wraps f so its error is logged at error level with %+v.
func (f TryFutureExtBacktrace[T]) LogErrWithBacktrace() LogErrorWithBacktraceFuture[T] {
	return LogErrorWithBacktraceFuture[T]{f: f, level: slog.LevelError, location: CallerLocation(1)}
}

// LogTrackedErrWithBacktrace wraps f so its error is logged at error level with %+v and location.
func (f TryFutureExtBacktrace[T]) LogTrackedErrWithBacktrace(location Location) LogErrorWithBacktraceFuture[T] {
	return LogErrorWithBacktraceFuture[T]{f: f, level: slog.LevelError, location: location}
}

// LogErrorFuture runs a fallible body and logs its error.
type LogErrorFuture[T any] struct {
	f        func() (T, error)
	level    slog.Level
	location Location
}

// Run runs the body and returns its value, or logs the error and returns false.
func (l LogErrorFuture[T]) Run() (T, bool) {
	output, err := l.f()
	if err != nil {
		logErrorWithCaller(l.location, err.Error(), l.level)
		var zero T
		return zero, false
	}
	return output, true
}

// LogErrorWithBacktraceFuture runs a fallible body and logs its error with %+v.
type LogErrorWithBacktraceFuture[T any] struct {
	f        func() (T, error)
	level    slog.Level
	location Location
}

// Run runs the body and returns its value, or logs the error and returns false.
func (l LogErrorWithBacktraceFuture[T]) Run() (T, bool) {
	output, err := l.f()
	if err != nil {
		logErrorWithCaller(l.location, fmt.Sprintf("%+v", err), l.level)
		var zero T
		return zero, false
	}
	return output, true
}

// UnwrapFuture runs a fallible body and panics on its error.
type UnwrapFuture[T any] struct {
	f func() (T, error)
}

// Run runs the body and returns its value, panicking on error.
func (u UnwrapFuture[T]) Run() T {
	output, err := u.f()
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %+v", err))
	}
	return output
}

// Deferred runs a function when Run is called unless Abort was called first.
type Deferred struct {
	f func()
}

// Abort drops the deferred function without running it.
func (d *Deferred) Abort() {
	d.f = nil
}

// Run runs the deferred function once, unless it was aborted.
func (d *Deferred) Run() {
	if f := d.f; f != nil {
		d.f = nil
		f()
	}
}

// Defer returns a Deferred that runs f; use it as `d := Defer(f); defer d.Run()`.
func Defer(f func()) Deferred {
	return Deferred{f: f}
}

// TypeIDHashBuilder builds TypeIDHasher values.
type TypeIDHashBuilder struct{}

// BuildHasher returns a new TypeIDHasher.
func (TypeIDHashBuilder) BuildHasher() TypeIDHasher {
	return TypeIDHasher{}
}

// TypeIDHasher hashes a type identity by keeping its first 8 bytes.
type TypeIDHasher struct {
	value uint64
}

// Write records the first 8 bytes of bytes.
func (h *TypeIDHasher) Write(bytes []byte) (int, error) {
	if len(bytes) >= 8 {
		h.value = binary.NativeEndian.Uint64(bytes[:8])
	} else {
		DebugPanic("expected a 64-bit value, did you use this hasher with something other than a TypeId?")
	}
	return len(bytes), nil
}

// Finish returns the hash value.
func (h *TypeIDHasher) Finish() uint64 {
	return h.value
}

// TruncateToBottomNSortedBy keeps the limit smallest items under compare, sorted.
func TruncateToBottomNSortedBy[T any](items *[]T, limit int, compare func(a, b T) int) {
	if limit == 0 {
		*items = (*items)[:0]
	}
	if len(*items) <= limit {
		slices.SortStableFunc(*items, compare)
		return
	}
	selectNthUnstableBy(*items, limit, compare)
	*items = (*items)[:limit]
	slices.SortStableFunc(*items, compare)
}

func selectNthUnstableBy[T any](s []T, n int, compare func(a, b T) int) {
	lo, hi := 0, len(s)-1
	for lo < hi {
		mid := lo + (hi-lo)/2
		if compare(s[mid], s[lo]) < 0 {
			s[mid], s[lo] = s[lo], s[mid]
		}
		if compare(s[hi], s[lo]) < 0 {
			s[hi], s[lo] = s[lo], s[hi]
		}
		if compare(s[hi], s[mid]) < 0 {
			s[hi], s[mid] = s[mid], s[hi]
		}
		pivot := s[mid]
		i, j := lo, hi
		for i <= j {
			for compare(s[i], pivot) < 0 {
				i++
			}
			for compare(s[j], pivot) > 0 {
				j--
			}
			if i <= j {
				s[i], s[j] = s[j], s[i]
				i++
				j--
			}
		}
		switch {
		case n <= j:
			hi = j
		case n >= i:
			lo = i
		default:
			return
		}
	}
}
