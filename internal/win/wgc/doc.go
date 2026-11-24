//go:build windows

// Package wgc ports the parts of the windows-capture crate that zed-scap's Windows engine uses:
// Windows.Graphics.Capture items for monitors and windows, the frame pool and session, the
// FrameArrived and Closed handlers as Go COM delegates, staging-texture frame readback and the
// capture thread with its dispatcher queue and message loop.
package wgc
