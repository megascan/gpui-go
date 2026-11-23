// Package gpui is a Go port of GPUI, the hybrid immediate and retained mode,
// GPU-accelerated UI framework of the Zed editor.
//
// Applications start with Application, create entities with App.New and
// open windows whose root views render trees of elements every frame. The
// platform packages under platform/ provide the windowing back ends; the root
// package itself imports no platform or renderer code.
package gpui
