//go:build !ios

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Build this example for an iOS target and package it with build-simulator.sh.")
}
