package scheduler

import (
	"errors"
	"strings"
	"testing"
)

func TestSecurity_SEC037_BackgroundPanicResolvesWithError(t *testing.T) {
	s := newFIFOScheduler()
	task := s.background().Spawn(func(*Scope) (int, error) {
		var b []int
		return b[len(b)+3], nil
	})
	propagated := func() (r any) {
		defer func() { r = recover() }()
		s.run()
		return nil
	}()
	if propagated != nil {
		t.Fatalf("background task panic propagated out of the scheduler: %v", propagated)
	}
	v, err := awaitTask(t, s, task)
	var pe *PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("background panic resolved to (%v, %v), want a *PanicError", v, err)
	}
	if errors.Is(err, ErrCanceled) {
		t.Fatalf("panic error must not match ErrCanceled: %v", err)
	}
	if !strings.Contains(pe.Error(), "index out of range") || len(pe.Stack) == 0 {
		t.Fatalf("panic error lacks value or stack: %q", pe.Error())
	}

	later := s.background().Spawn(func(*Scope) (int, error) { return 7, nil })
	s.run()
	if v, err := awaitTask(t, s, later); v != 7 || err != nil {
		t.Fatalf("task after a panicked one: (%v, %v)", v, err)
	}
}

func TestSecurity_SEC037_BackgroundCancelStillUnwinds(t *testing.T) {
	s := newFIFOScheduler()
	unwound := false
	task := s.background().Spawn(func(cx *Scope) (int, error) {
		defer func() { unwound = true }()
		y := Yield(1)
		Await[struct{}](cx, &y)
		Await[struct{}](cx, PollFn(func(*Context) (struct{}, bool) { return struct{}{}, false }))
		return 1, nil
	})
	s.run()
	task.Cancel()
	s.run()
	if !unwound {
		t.Fatal("canceled background body did not unwind")
	}
}

func TestSecurity_SEC037_ForegroundPanicStillPropagates(t *testing.T) {
	s := newFIFOScheduler()
	s.foreground().Spawn(func(*Scope) (int, error) { panic("boom") }).Detach()
	got := func() (r any) {
		defer func() { r = recover() }()
		s.run()
		return nil
	}()
	if got != "boom" {
		t.Fatalf("foreground panic: got %v, want boom", got)
	}
}

func TestSecurity_R2_13_RealtimeAndDedicatedPanicsSurfaceAsErrors(t *testing.T) {
	s := newFIFOScheduler()
	var panicErr *PanicError
	rt := s.background().SpawnRealtime(func(*Scope) (int, error) { panic("rt") }).Fallible()
	if r := BlockOn[Result[int]](rt); !errors.As(r.Err, &panicErr) {
		t.Fatalf("realtime panic: %v", r.Err)
	}
	dd := s.background().SpawnDedicated(func(LocalExecutor, *Scope) (int, error) { panic("dd") }).Fallible()
	if r := BlockOn[Result[int]](dd); !errors.As(r.Err, &panicErr) {
		t.Fatalf("dedicated panic: %v", r.Err)
	}
	de := NewDedicatedExecutor(s.background())
	defer de.Close()
	task := de.Spawn(func(*Scope) (int, error) { panic("de") }).Fallible()
	if r := BlockOn[Result[int]](task); !errors.As(r.Err, &panicErr) {
		t.Fatalf("dedicated executor panic: %v", r.Err)
	}
	after := de.Spawn(func(*Scope) (int, error) { return 7, nil }).Fallible()
	if r := BlockOn[Result[int]](after); r.Err != nil || r.Value != 7 {
		t.Fatalf("dedicated executor after panic: %v", r)
	}
}
