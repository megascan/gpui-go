package gpui

import (
	"errors"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

func TestSecurity_R2_13_ScopedSurfacesChildPanic(t *testing.T) {
	dispatcher := NewTestDispatcher(0)
	background := NewBackgroundExecutor(dispatcher)
	foreground := NewForegroundExecutor(dispatcher)
	siblingRan := false
	var scopedErr error
	task := foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		scopedErr = background.Scoped(s, func(scope *Scope) {
			scope.Spawn(func(*scheduler.Scope) { panic("scoped boom") })
			scope.Spawn(func(*scheduler.Scope) { siblingRan = true })
		})
		return struct{}{}, nil
	})
	dispatcher.RunUntilParked()
	if !task.IsReady() {
		t.Fatal("scoped task did not complete after a child panicked")
	}
	task.Detach()
	if !siblingRan {
		t.Fatal("sibling of the panicking scoped task did not run")
	}
	var panicErr *scheduler.PanicError
	if !errors.As(scopedErr, &panicErr) {
		t.Fatalf("Scoped returned %v, want a *scheduler.PanicError", scopedErr)
	}
	if panicErr.Value != "scoped boom" {
		t.Fatalf("PanicError.Value = %v, want scoped boom", panicErr.Value)
	}
	var cleanErr error
	clean := foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		cleanErr = background.Scoped(s, func(scope *Scope) {
			scope.Spawn(func(*scheduler.Scope) {})
		})
		return struct{}{}, nil
	})
	dispatcher.RunUntilParked()
	if !clean.IsReady() {
		t.Fatal("clean scoped task did not complete")
	}
	clean.Detach()
	if cleanErr != nil {
		t.Fatalf("Scoped without panics returned %v", cleanErr)
	}
}
