//go:build deps

// Package deps pins every third-party module the port uses so that module
// pruning keeps them in go.mod before the packages that import them exist.
package deps

import (
	_ "github.com/gen2brain/avif"
	_ "github.com/go-text/typesetting/font"
	_ "github.com/go-text/typesetting/shaping"
	_ "github.com/godbus/dbus/v5"
	_ "github.com/jezek/xgb"
	_ "github.com/jezek/xgbutil"
	_ "github.com/xo/okage"
	_ "github.com/xo/okage/svg"
	_ "golang.org/x/image/webp"
	_ "golang.org/x/sys/cpu"
)
