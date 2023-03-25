// Copyright (C) 2023 Storx Labs, Inc.
// See LICENSE for copying information.

package server

import (
	"syscall"

	"go.uber.org/zap"
)

const tcpFastOpenServer = 15

func setTCPFastOpen(fd uintptr, queue int) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_TCP, tcpFastOpenServer, 1)
}

func tryInitFastOpen(*zap.Logger) {
	// should we log or check something along the lines of
	// netsh int tcp set global fastopen=enabled
	// netsh int tcp set global fastopenfallback=disabled
	// ?
}
