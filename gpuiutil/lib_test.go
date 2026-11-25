package gpuiutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"unsafe"
)

func TestGpuiUtil_TypeIDHasher(t *testing.T) {
	verifyHashingWith := func(typeID reflect.Type) {
		type iface struct {
			tab  unsafe.Pointer
			data unsafe.Pointer
		}
		data := uintptr((*iface)(unsafe.Pointer(&typeID)).data)
		bytes := unsafe.Slice((*byte)(unsafe.Pointer(&data)), unsafe.Sizeof(data))
		hasher := TypeIDHashBuilder{}.BuildHasher()
		_, _ = hasher.Write(bytes)
		if hasher.Finish() == 0 {
			t.Fatalf("hash of %v is 0", typeID)
		}
	}
	verifyHashingWith(reflect.TypeFor[uint]())
	verifyHashingWith(reflect.TypeFor[struct{}]())
	verifyHashingWith(reflect.TypeFor[string]())
	verifyHashingWith(reflect.TypeFor[*string]())
	verifyHashingWith(reflect.TypeFor[[]uint8]())
}

type loggedRecord struct {
	line    int
	message string
}

type testLogger struct {
	t       *testing.T
	file    string
	records *[]loggedRecord
}

func (h testLogger) Enabled(context.Context, slog.Level) bool { return true }

func (h testLogger) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]slog.Value{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value
		return true
	})
	if got := attrs["target"].String(); got != "gpuiutil::lib_test" {
		h.t.Errorf("target = %q", got)
	}
	if got := attrs["module_path"].String(); got != "github.com/megascan/gpui-go/gpuiutil/lib_test.go" {
		h.t.Errorf("module_path = %q", got)
	}
	if got := attrs["file"].String(); got != h.file {
		h.t.Errorf("file = %q, want %q", got, h.file)
	}
	if r.Level != slog.LevelError {
		h.t.Errorf("level = %v", r.Level)
	}
	*h.records = append(*h.records, loggedRecord{int(attrs["line"].Int64()), r.Message})
	return nil
}

func (h testLogger) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h testLogger) WithGroup(string) slog.Handler      { return h }

func TestGpuiUtilLoggingTests_LoggingPreservesDiagnosticsAndCallers(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	var records []loggedRecord
	prev := slog.Default()
	slog.SetDefault(slog.New(testLogger{t: t, file: file, records: &records}))
	defer slog.SetDefault(prev)
	assertLogged := func(line int, message string) {
		t.Helper()
		want := []loggedRecord{{line, message}}
		if !slices.Equal(records, want) {
			t.Fatalf("records = %v, want %v", records, want)
		}
		records = nil
	}
	line := func() int { _, _, l, _ := runtime.Caller(1); return l }

	err := fmt.Errorf("outer context: %w", fmt.Errorf("inner context: %w", errors.New("root failure")))
	display := "outer context: inner context: root failure"
	debugText := fmt.Sprintf("%+v", err)
	l := line() + 1
	if _, ok := Result(struct{}{}, err).LogErr(); ok {
		t.Fatal("LogErr returned a value")
	}
	assertLogged(l, display)
	l = line() + 1
	if _, ok := Result(struct{}{}, err).LogErrWithBacktrace(); ok {
		t.Fatal("LogErrWithBacktrace returned a value")
	}
	assertLogged(l, debugText)
	ready := func() (struct{}, error) { return struct{}{}, err }
	l = line() + 1
	future := TryFutureExt[struct{}](ready).LogErr()
	if _, ok := future.Run(); ok {
		t.Fatal("LogErrorFuture returned a value")
	}
	assertLogged(l, display)
	l = line() + 1
	futureBt := TryFutureExtBacktrace[struct{}](ready).LogErrWithBacktrace()
	if _, ok := futureBt.Run(); ok {
		t.Fatal("LogErrorWithBacktraceFuture returned a value")
	}
	assertLogged(l, debugText)
}

func TestGpuiUtil_Deferred(t *testing.T) {
	ran := 0
	func() {
		d := Defer(func() { ran++ })
		defer d.Run()
	}()
	func() {
		d := Defer(func() { ran += 10 })
		defer d.Run()
		d.Abort()
	}()
	if ran != 1 {
		t.Fatalf("ran = %d", ran)
	}
}

func TestGpuiUtil_TruncateToBottomNSortedBy(t *testing.T) {
	cmp := func(a, b int) int { return a - b }
	for limit := range 12 {
		items := []int{5, 3, 9, 1, 7, 2, 8, 6, 4, 0}
		TruncateToBottomNSortedBy(&items, limit, cmp)
		want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}[:min(limit, 10)]
		if !slices.Equal(items, want) {
			t.Fatalf("limit %d: %v", limit, items)
		}
	}
}

func TestGpuiUtil_MiscHelpers(t *testing.T) {
	if !IsUTF8CharBoundary('a') || IsUTF8CharBoundary(0x80) || !IsUTF8CharBoundary(0xC0) {
		t.Fatal("IsUTF8CharBoundary")
	}
	v := 3
	if PostInc(&v) != 3 || v != 4 {
		t.Fatal("PostInc")
	}
	var negZero float32
	negZero = -negZero
	if CanonicalFloatBits(negZero) != CanonicalFloatBits(0) || CanonicalFloatBits(1) != 0x3f800000 {
		t.Fatal("CanonicalFloatBits")
	}
}

func TestGpuiUtil_LogErrNilIsOk(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	var records []loggedRecord
	prev := slog.Default()
	slog.SetDefault(slog.New(testLogger{t: t, file: file, records: &records}))
	defer slog.SetDefault(prev)
	LogErr(nil)
	if len(records) != 0 {
		t.Fatalf("LogErr(nil) logged %v", records)
	}
}
