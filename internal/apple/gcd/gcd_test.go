//go:build darwin

package gcd

import (
	"sync"
	"testing"
	"time"
)

func TestGcd_Symbols(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if MainQueue() == 0 {
		t.Fatal("_dispatch_main_q not resolved")
	}
	for _, p := range []QueuePriority{QueuePriorityHigh, QueuePriorityDefault, QueuePriorityLow, QueuePriorityBackground} {
		if GlobalQueue(p) == 0 {
			t.Errorf("global queue of priority %d is nil", p)
		}
	}
	for _, q := range []QoSClass{QoSClassUserInteractive, QoSClassUserInitiated, QoSClassDefault, QoSClassUtility, QoSClassBackground} {
		if GlobalQueueQoS(q) == 0 {
			t.Errorf("global queue of QoS %#x is nil", q)
		}
	}
	if GlobalQueue(QueuePriorityDefault) != GlobalQueueQoS(QoSClassDefault) {
		t.Error("default priority and default QoS name different queues")
	}
	if After(time.Second) <= After(0) {
		t.Error("dispatch_time did not advance")
	}
}

func TestGcd_AsyncRunsOffMainThread(t *testing.T) {
	done := make(chan bool, 1)
	GlobalQueue(QueuePriorityHigh).Async(func() { done <- IsMainThread() })
	select {
	case onMain := <-done:
		if onMain {
			t.Fatal("global queue work ran on the main thread")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("async work did not run")
	}
}

func TestGcd_ExecAfterF(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	start := time.Now()
	var got uintptr
	fn := NewFunction(func(context uintptr) {
		got = context
		wg.Done()
	})
	ExecAfterF(After(20*time.Millisecond), GlobalQueue(QueuePriorityDefault), 7, fn)
	wg.Wait()
	if got != 7 {
		t.Fatalf("context = %d, want 7", got)
	}
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("dispatch_after_f fired early")
	}
}

func TestGcd_SpawnThread(t *testing.T) {
	done := make(chan uint64, 1)
	if err := SpawnThread(func() {
		if IsMainThread() {
			done <- 0
			return
		}
		done <- CurrentThreadID()
	}); err != nil {
		t.Fatal(err)
	}
	id := <-done
	if id == 0 || id == CurrentThreadID() {
		t.Fatalf("spawned thread id = %d", id)
	}
	if CurrentMachThread() == 0 {
		t.Fatal("mach thread port is zero")
	}
}

func TestGcd_Handles(t *testing.T) {
	var h Handles[string]
	a, b := h.Insert("a"), h.Insert("b")
	if a == 0 || a == b {
		t.Fatalf("keys %d, %d", a, b)
	}
	if v, ok := h.Get(a); !ok || v != "a" {
		t.Fatal("Get failed")
	}
	if v, ok := h.Take(a); !ok || v != "a" || h.Len() != 1 {
		t.Fatal("Take failed")
	}
	if _, ok := h.Take(a); ok {
		t.Fatal("second Take succeeded")
	}
}
