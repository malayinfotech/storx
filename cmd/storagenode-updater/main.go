// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

//go:build !windows
// +build !windows

package main

import (
	"go.uber.org/zap"

	"private/process"
)

func main() {
	loggerFunc := func(logger *zap.Logger) *zap.Logger {
		return logger.With(zap.String("Process", updaterServiceName))
	}

	process.ExecWithCustomConfigAndLogger(rootCmd, true, process.LoadConfig, loggerFunc)
}
