//go:build windows

// Package windows is the Windows platform: Win32 windows and input, DirectX rendering,
// DirectWrite text and the shell integrations (gpui_windows).
package windows

import "github.com/megascan/gpui-go/internal/win/win32"

// HWND is a window handle.
type HWND = win32.HWND
