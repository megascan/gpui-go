//go:build js && wasm

// Package web is GPUI's browser platform. It uses one document-owned canvas and supports one
// top-level window. Browser WebGPU is preferred by default, with an automatic WebGL2 fallback.
// Applications can force either backend with WebBackendPreference. Opening a second top-level
// window, or reopening one after it closes, returns a WebWindowError.
package web

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"syscall/js"

	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// WebBackendPreference selects the browser graphics API.
type WebBackendPreference = wgpu.WebBackendPreference

const (
	WebBackendPreferenceAuto   = wgpu.WebBackendPreferenceAuto
	WebBackendPreferenceWebGpu = wgpu.WebBackendPreferenceWebGpu
	WebBackendPreferenceWebGl  = wgpu.WebBackendPreferenceWebGl
)

type jsPanicError struct {
	value any
}

func (e jsPanicError) Error() string {
	return fmt.Sprint(e.value)
}

func jsTry(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
				return
			}
			err = jsPanicError{r}
		}
	}()
	f()
	return nil
}

func jsCall(target js.Value, method string, args ...any) (result js.Value, err error) {
	err = jsTry(func() {
		result = target.Call(method, args...)
	})
	return result, err
}

func jsNew(constructor js.Value, args ...any) (result js.Value, err error) {
	err = jsTry(func() {
		result = constructor.New(args...)
	})
	return result, err
}

func jsGet(target js.Value, property string) js.Value {
	if !jsIsObject(target) {
		return js.Undefined()
	}
	var result js.Value
	if jsTry(func() { result = target.Get(property) }) != nil {
		return js.Undefined()
	}
	return result
}

func jsIsObject(v js.Value) bool {
	t := v.Type()
	return t == js.TypeObject || t == js.TypeFunction
}

func jsIsNullish(v js.Value) bool {
	return v.IsUndefined() || v.IsNull()
}

func jsNumber(v js.Value) (float64, bool) {
	if v.Type() != js.TypeNumber {
		return 0, false
	}
	return v.Float(), true
}

func jsString(v js.Value) (string, bool) {
	if v.Type() != js.TypeString {
		return "", false
	}
	return v.String(), true
}

func jsBool(v js.Value) (bool, bool) {
	if v.Type() != js.TypeBoolean {
		return false, false
	}
	return v.Bool(), true
}

func jsHas(target js.Value, property string) bool {
	has, err := jsCall(js.Global().Get("Reflect"), "has", target, property)
	if err != nil {
		return false
	}
	b, _ := jsBool(has)
	return b
}

func jsInstanceOf(v js.Value, constructorName string) bool {
	constructor := js.Global().Get(constructorName)
	if constructor.Type() != js.TypeFunction || !jsIsObject(v) {
		return false
	}
	return v.InstanceOf(constructor)
}

func jsDebug(v js.Value) string {
	var text string
	if jsTry(func() { text = js.Global().Get("String").Invoke(v).String() }) != nil {
		text = v.Type().String()
	}
	return "JsValue(" + text + ")"
}

func jsErrorDebug(err error) string {
	var jsErr js.Error
	if errors.As(err, &jsErr) {
		return jsDebug(jsErr.Value)
	}
	return err.Error()
}

func jsFuncOnce(f func(args []js.Value)) js.Func {
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer fn.Release()
		f(args)
		return nil
	})
	return fn
}

type promiseOutcome struct {
	value    js.Value
	rejected bool
}

func promiseThen(promise js.Value, settle func(promiseOutcome)) error {
	var resolve, reject js.Func
	release := func() {
		resolve.Release()
		reject.Release()
	}
	resolve = js.FuncOf(func(this js.Value, args []js.Value) any {
		release()
		settle(promiseOutcome{value: argOrUndefined(args)})
		return nil
	})
	reject = js.FuncOf(func(this js.Value, args []js.Value) any {
		release()
		settle(promiseOutcome{value: argOrUndefined(args), rejected: true})
		return nil
	})
	if _, err := jsCall(promise, "then", resolve, reject); err != nil {
		release()
		return err
	}
	return nil
}

func argOrUndefined(args []js.Value) js.Value {
	if len(args) == 0 {
		return js.Undefined()
	}
	return args[0]
}

type jsRejection struct {
	value js.Value
}

func (e jsRejection) Error() string {
	return jsDebug(e.value)
}

func awaitPromise(cx scheduler.Awaiter, promise js.Value) (js.Value, error) {
	tx, rx := scheduler.Oneshot[promiseOutcome]()
	if err := promiseThen(promise, func(outcome promiseOutcome) { _ = tx.Send(outcome) }); err != nil {
		return js.Undefined(), err
	}
	outcome, err := rx.Recv(cx)
	if err != nil {
		return js.Undefined(), err
	}
	if outcome.rejected {
		return outcome.value, jsRejection{outcome.value}
	}
	return outcome.value, nil
}

func awaitPromiseBlocking(promise js.Value) (js.Value, error) {
	done := make(chan promiseOutcome, 1)
	if err := promiseThen(promise, func(outcome promiseOutcome) { done <- outcome }); err != nil {
		return js.Undefined(), err
	}
	outcome := <-done
	if outcome.rejected {
		return outcome.value, jsRejection{outcome.value}
	}
	return outcome.value, nil
}

func formatF32(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

func formatF64(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func saturatingU32(v float64) uint32 {
	switch {
	case math.IsNaN(v) || v <= 0:
		return 0
	case v >= math.MaxUint32:
		return math.MaxUint32
	}
	return uint32(v)
}

func browserWindow() js.Value {
	window := js.Global().Get("window")
	if jsIsNullish(window) {
		panic("must be running in a browser window context")
	}
	return window
}
