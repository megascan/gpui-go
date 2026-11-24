# M4-windows-directx-renderer debt

- `tryToRecoverFromDeviceLost` (directx_devices.go, M4-windows-d3d-atlas) sleeps with time.Sleep between retries as upstream's thread::sleep; HandleDeviceLost inherits it on the main thread.
- No end-to-end test creates a DirectXRenderer (needs an HWND); first real coverage comes with M4-windows-window and the Windows CI runner.
