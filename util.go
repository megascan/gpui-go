package gpui

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// Fluent is the FluentBuilder mixin: helpers for building objects with imperative conditionals
// in a fluent style. Embed it and bind it with BindFluent.
type Fluent[S any] struct {
	self S
}

// BindFluent binds a Fluent mixin to the element that embeds it.
func BindFluent[S any](m *Fluent[S], self S) {
	m.self = self
}

// Map imperatively modifies self with the given closure.
func (f *Fluent[S]) Map[U any](fn func(this S) U) U {
	return fn(f.self)
}

// When conditionally modifies self with the given closure.
func (f *Fluent[S]) When(condition bool, then func(this S) S) S {
	return f.Map[S](func(this S) S {
		if condition {
			return then(this)
		}
		return this
	})
}

// WhenElse modifies self with then when condition holds and with elseFn otherwise.
func (f *Fluent[S]) WhenElse(condition bool, then func(this S) S, elseFn func(this S) S) S {
	return f.Map[S](func(this S) S {
		if condition {
			return then(this)
		}
		return elseFn(this)
	})
}

// WhenSome modifies self with the option's value when it is some.
func (f *Fluent[S]) WhenSome[T any](option Option[T], then func(this S, value T) S) S {
	return f.Map[S](func(this S) S {
		if value, ok := option.Get(); ok {
			return then(this, value)
		}
		return this
	})
}

// WhenNone modifies self with the given closure when the option is none.
func (f *Fluent[S]) WhenNone[T any](option Option[T], then func(this S) S) S {
	return f.Map[S](func(this S) S {
		if option.IsSome() {
			return this
		}
		return then(this)
	})
}

// FutureWithTimeout requires future to complete before timeout elapses on executor's clock
// (upstream's FutureExt::with_timeout).
func FutureWithTimeout[T any](future scheduler.Future[T], timeout time.Duration, executor BackgroundExecutor) *WithTimeout[T] {
	return &WithTimeout[T]{future: future, timer: executor.Timer(timeout)}
}

// WithTimeout is a future that fails with Timeout when its timer fires before the inner future resolves.
type WithTimeout[T any] struct {
	future scheduler.Future[T]
	timer  *Task[struct{}]
}

// Poll polls the inner future, then the timer.
func (w *WithTimeout[T]) Poll(cx *scheduler.Context) (scheduler.Result[T], bool) {
	if output, ok := w.future.Poll(cx); ok {
		return scheduler.Result[T]{Value: output}, true
	}
	if _, ok := w.timer.Poll(cx); ok {
		return scheduler.Result[T]{Err: Timeout{}}, true
	}
	return scheduler.Result[T]{}, false
}

// Timeout is the error returned by WithTimeout when the timeout elapsed before the future resolved.
type Timeout struct{}

// Error formats the error.
func (Timeout) Error() string {
	return "Timed out before future resolved"
}

func atomicIncrIfNotZero(counter *atomic.Uint64) uint64 {
	loaded := counter.Load()
	for {
		if loaded == 0 {
			return 0
		}
		if counter.CompareAndSwap(loaded, loaded+1) {
			return loaded + 1
		}
		loaded = counter.Load()
	}
}

func roundHalfTowardZero(value float32) float32 {
	a := float32(math.Abs(float64(value))) - 0.5
	return float32(math.Copysign(math.Ceil(float64(a)), float64(value)))
}

func roundHalfTowardZeroF64(value float64) float64 {
	return math.Copysign(math.Ceil(math.Abs(value)-0.5), value)
}

func roundToDevicePixel(logical, scaleFactor float32) float32 {
	return roundHalfTowardZero(logical * scaleFactor)
}

func roundStrokeToDevicePixel(logical, scaleFactor float32) float32 {
	if logical == 0 {
		return 0
	}
	return max(roundToDevicePixel(max(logical, 0), scaleFactor), 1)
}

func floorToDevicePixel(logical, scaleFactor float32) float32 {
	return float32(math.Floor(float64(logical * scaleFactor)))
}

func ceilToDevicePixel(logical, scaleFactor float32) float32 {
	return float32(math.Ceil(float64(logical * scaleFactor)))
}
