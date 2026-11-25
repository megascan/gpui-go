//go:build windows || js

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "shoot: only Linux (Hyprland) is supported")
	os.Exit(1)
}
