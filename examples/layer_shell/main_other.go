//go:build !linux

package main

func exampleMain() {
	panic("This example requires the `wayland` feature and a linux system.")
}
