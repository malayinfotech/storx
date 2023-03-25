// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package version

import _ "unsafe" // needed for go:linkname

//go:linkname buildTimestamp storx/private/version.buildTimestamp
var buildTimestamp string

//go:linkname buildCommitHash storx/private/version.buildCommitHash
var buildCommitHash string

//go:linkname buildVersion storx/private/version.buildVersion
var buildVersion string

//go:linkname buildRelease storx/private/version.buildRelease
var buildRelease string

// ensure that linter understands that the variables are being used.
func init() { use(buildTimestamp, buildCommitHash, buildVersion, buildRelease) }

func use(...interface{}) {}
