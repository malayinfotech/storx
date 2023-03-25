// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information

// Package testplanet implements full network wiring for testing.
//
// testplanet provides access to most of the internals of satellites,
// storagenodes and uplinks.
//
// # Database
//
// It does require setting two variables for the databases:
//
//	STORX_TEST_POSTGRES=postgres://storx:storx-pass@test-postgres/teststorx?sslmode=disable
//	STORX_TEST_COCKROACH=cockroach://root@localhost:26257/master?sslmode=disable
//
// When you wish to entirely omit either of them from the test output, it's possible to use:
//
//	STORX_TEST_POSTGRES=omit
//	STORX_TEST_COCKROACH=omit
//
// # Host
//
// It's possible to change the listing host with:
//
//	STORX_TEST_HOST=127.0.0.2;127.0.0.3
//
// # Debugging
//
// For debugging, it's possible to set STORX_TEST_MONKIT to get a trace per test.
//
//	STORX_TEST_MONKIT=svg
//	STORX_TEST_MONKIT=json
//
// By default, it saves the output the same folder as the test. However, if you wish
// to specify a separate folder, you can specify an absolute directory:
//
//	STORX_TEST_MONKIT=svg,dir=/home/user/debug/trace
//
// Note, due to how go tests work, it's not possible to specify a relative directory.
package testplanet
