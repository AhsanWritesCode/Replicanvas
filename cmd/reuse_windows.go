//go:build windows

package main

import "syscall"

// Works for Windows

const SO_EXCLUSIVEADDRUSE = ^syscall.SO_REUSEADDR

// setReuseAddr configures a platform-appropriate socket reuse option so the
// server can restart and rebind the port more reliably after shutdown.
func setReuseAddr(network, address string, conn syscall.RawConn) error {
	return conn.Control(func(fd uintptr) {
		syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, SO_EXCLUSIVEADDRUSE, 1)
	})
}
