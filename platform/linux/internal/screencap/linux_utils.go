//go:build linux

package screencap

func linuxIsSupported() bool { return true }

func linuxHasPermission() bool { return true }
