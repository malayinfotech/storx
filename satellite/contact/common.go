// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package contact

import (
	"github.com/jtolio/eventkit"
	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
)

var (
	// Error is the default error class for contact package.
	Error = errs.Class("contact")

	mon = monkit.Package()

	ek = eventkit.Package()
)
