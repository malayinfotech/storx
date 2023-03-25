// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"

	"common/rpc/rpcpool"
	"storx/cmd/uplink/ulext"
	"storx/cmd/uplink/ulfs"
	"uplink"
	privateAccess "uplink/private/access"
	"uplink/private/transport"
)

const uplinkCLIUserAgent = "uplink-cli"

func (ex *external) OpenFilesystem(ctx context.Context, accessName string, options ...ulext.Option) (ulfs.Filesystem, error) {
	project, err := ex.OpenProject(ctx, accessName, options...)
	if err != nil {
		return nil, err
	}
	return ulfs.NewMixed(ulfs.NewLocal(ulfs.NewLocalBackendOS()), ulfs.NewRemote(project)), nil
}

func (ex *external) OpenProject(ctx context.Context, accessName string, options ...ulext.Option) (*uplink.Project, error) {
	opts := ulext.LoadOptions(options...)

	access, err := ex.OpenAccess(accessName)
	if err != nil {
		return nil, err
	}

	if opts.EncryptionBypass {
		if err := privateAccess.EnablePathEncryptionBypass(access); err != nil {
			return nil, err
		}
	}

	config := uplink.Config{
		UserAgent: uplinkCLIUserAgent,
	}

	if opts.ConnectionPoolOptions != (rpcpool.Options{}) {
		if err := transport.SetConnectionPool(ctx, &config, rpcpool.New(opts.ConnectionPoolOptions)); err != nil {
			return nil, err
		}
	}

	return config.OpenProject(ctx, access)
}
