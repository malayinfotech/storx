// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"

	"github.com/zeebo/clingy"
	"github.com/zeebo/errs"

	"storx/cmd/uplink/ulext"
)

type cmdAccessExport struct {
	ex ulext.External

	name     string
	filename string
}

func newCmdAccessExport(ex ulext.External) *cmdAccessExport {
	return &cmdAccessExport{ex: ex}
}

func (c *cmdAccessExport) Setup(params clingy.Parameters) {
	c.name = params.Arg("name", "Name of the access to export").(string)
	c.filename = params.Arg("filename", "Name of the file to save to").(string)
}

func (c *cmdAccessExport) Execute(ctx context.Context) error {
	if c.filename == "" {
		return errs.New("Must specify a filename to write to.")
	}

	access, err := c.ex.OpenAccess(c.name)
	if err != nil {
		return err
	}

	return c.ex.ExportAccess(ctx, access, c.filename)
}
