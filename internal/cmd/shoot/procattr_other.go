//go:build !linux && !windows && !js

package main

import "syscall"

func childProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
