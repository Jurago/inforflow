//go:build !linux

package main

import "fmt"

func syscallGetSockOptRcvBuf(fd int) (int, error) {
	return 0, fmt.Errorf("SO_RCVBUF query unsupported")
}
