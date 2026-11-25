//go:build !darwin || ios

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Native views are only composited on macOS.")
}
