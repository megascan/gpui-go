//go:build windows

// Package winmanifest links upstream gpui's application manifest (gpui/resources/windows/gpui.manifest.xml:
// per-monitor-v2 DPI awareness, common controls v6, Windows 10 supportedOS, segment heap) into every
// Windows binary that imports platform/windows, as gpui's default `windows-manifest` feature does.
// Regenerate the .syso files with gen.sh (llvm-windres). Build with -tags gpui_no_manifest to ship
// your own manifest instead.
package winmanifest

//go:generate sh gen.sh
