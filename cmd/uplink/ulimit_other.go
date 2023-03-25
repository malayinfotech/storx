// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

//go:build !linux && !darwin && !freebsd
// +build !linux,!darwin,!freebsd

package main

func raiseUlimits() {}
