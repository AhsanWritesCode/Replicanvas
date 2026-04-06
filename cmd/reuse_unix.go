//go:build !windows

package main

import "syscall"

// Works for UNIX Mac systems

// setReuseAddr configures a platform-appropriate socket reuse option so the
// server can restart and rebind the port more reliably after shutdown.
func setReuseAddr(network, address string, conn syscall.RawConn) error {
	return conn.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
}
