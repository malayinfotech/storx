// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"

	"github.com/zeebo/clingy"

	"storx/cmd/uplink/ulext"
)

type cmdAccessRevoke struct {
	ex ulext.External

	access  string
	revokee string
}

func newCmdAccessRevoke(ex ulext.External) *cmdAccessRevoke {
	return &cmdAccessRevoke{ex: ex}
}

func (c *cmdAccessRevoke) Setup(params clingy.Parameters) {
	c.access = params.Flag("access", "Access name or value performing the revoke", "").(string)
	c.revokee = params.Arg("revokee", "Access name or value revoke").(string)
}

func (c *cmdAccessRevoke) Execute(ctx context.Context) error {
	project, err := c.ex.OpenProject(ctx, c.access)
	if err != nil {
		return err
	}
	defer func() { _ = project.Close() }()

	access, err := c.ex.OpenAccess(c.revokee)
	if err != nil {
		return err
	}

	if err := project.RevokeAccess(ctx, access); err != nil {
		return err
	}

	fmt.Fprintf(clingy.Stdout(ctx), "Revoked access %q\n", c.revokee)

	return nil
}