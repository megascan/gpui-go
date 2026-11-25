//go:build ios && cgo

package main

import "C"

//export gpui_ios_main
func gpui_ios_main() {
	runExample()
}
