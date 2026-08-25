//go:build linux

package main

import "syscall"

func syscallGetSockOptRcvBuf(fd int) (int, error) {
	return syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF)
}
