//go:build unix

package gpui

import "syscall"

const imageOpenNonblockFlag = syscall.O_NONBLOCK
