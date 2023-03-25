// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package storage

import (
	"context"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/rpc"
	"common/storx"
	"storx/multinode/nodes"
	"storx/private/multinodepb"
)

var (
	mon = monkit.Package()
	// Error is an error class for storage service error.
	Error = errs.Class("storage")
)

// Service exposes all storage related logic.
//
// architecture: Service
type Service struct {
	log    *zap.Logger
	dialer rpc.Dialer
	nodes  nodes.DB
}

// NewService creates new instance of Service.
func NewService(log *zap.Logger, dialer rpc.Dialer, nodes nodes.DB) *Service {
	return &Service{
		log:    log,
		dialer: dialer,
		nodes:  nodes,
	}
}

// Usage retrieves node's daily storage usage for provided interval.
func (service *Service) Usage(ctx context.Context, nodeID storx.NodeID, from, to time.Time) (_ Usage, err error) {
	defer mon.Task()(&ctx)(&err)

	node, err := service.nodes.Get(ctx, nodeID)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	usage, err := service.dialUsage(ctx, node, from, to)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	return usage, nil
}

// UsageSatellite retrieves node's daily storage usage for provided interval and satellite.
func (service *Service) UsageSatellite(ctx context.Context, nodeID, satelliteID storx.NodeID, from, to time.Time) (_ Usage, err error) {
	defer mon.Task()(&ctx)(&err)

	node, err := service.nodes.Get(ctx, nodeID)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	usage, err := service.dialUsageSatellite(ctx, node, satelliteID, from, to)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	return usage, nil
}

// TotalUsage retrieves aggregated daily storage usage for provided interval.
func (service *Service) TotalUsage(ctx context.Context, from, to time.Time) (_ Usage, err error) {
	defer mon.Task()(&ctx)(&err)

	nodesList, err := service.nodes.List(ctx)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	var totalSummary float64
	cache := make(UsageStampDailyCache)

	for _, node := range nodesList {
		usage, err := service.dialUsage(ctx, node, from, to)
		if err != nil {
			if nodes.ErrNodeNotReachable.Has(err) {
				continue
			}

			return Usage{}, Error.Wrap(err)
		}

		totalSummary += usage.Summary
		for _, stamp := range usage.Stamps {
			cache.Add(stamp)
		}
	}

	return Usage{
		Stamps:  cache.Sorted(),
		Summary: totalSummary,
	}, nil
}

// TotalUsageSatellite retrieves aggregated daily storage usage for provided interval and satellite.
func (service *Service) TotalUsageSatellite(ctx context.Context, satelliteID storx.NodeID, from, to time.Time) (_ Usage, err error) {
	defer mon.Task()(&ctx)(&err)

	nodesList, err := service.nodes.List(ctx)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	var totalSummary float64
	cache := make(UsageStampDailyCache)

	for _, node := range nodesList {
		usage, err := service.dialUsageSatellite(ctx, node, satelliteID, from, to)
		if err != nil {
			if nodes.ErrNodeNotReachable.Has(err) {
				continue
			}

			return Usage{}, Error.Wrap(err)
		}

		totalSummary += usage.Summary
		for _, stamp := range usage.Stamps {
			cache.Add(stamp)
		}
	}

	return Usage{
		Stamps:  cache.Sorted(),
		Summary: totalSummary,
	}, nil
}

// TotalDiskSpace returns all info about all storagenodes disk space usage.
func (service *Service) TotalDiskSpace(ctx context.Context) (totalDiskSpace DiskSpace, err error) {
	defer mon.Task()(&ctx)(&err)

	listNodes, err := service.nodes.List(ctx)
	if err != nil {
		return DiskSpace{}, Error.Wrap(err)
	}

	for _, node := range listNodes {
		diskSpace, err := service.dialDiskSpace(ctx, node)
		if err != nil {
			if nodes.ErrNodeNotReachable.Has(err) {
				continue
			}

			return DiskSpace{}, Error.Wrap(err)
		}

		totalDiskSpace.Add(diskSpace)
	}

	return totalDiskSpace, nil
}

// DiskSpace returns all info about concrete storagenode disk space usage.
func (service *Service) DiskSpace(ctx context.Context, nodeID storx.NodeID) (_ DiskSpace, err error) {
	defer mon.Task()(&ctx)(&err)

	node, err := service.nodes.Get(ctx, nodeID)
	if err != nil {
		return DiskSpace{}, Error.Wrap(err)
	}

	return service.dialDiskSpace(ctx, node)
}

// dialDiskSpace dials node and retrieves all info about concrete storagenode disk space usage.
func (service *Service) dialDiskSpace(ctx context.Context, node nodes.Node) (diskSpace DiskSpace, err error) {
	conn, err := service.dialer.DialNodeURL(ctx, storx.NodeURL{
		ID:      node.ID,
		Address: node.PublicAddress,
	})
	if err != nil {
		return DiskSpace{}, nodes.ErrNodeNotReachable.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, conn.Close())
	}()

	storageClient := multinodepb.NewDRPCStorageClient(conn)

	diskSpaceResponse, err := storageClient.DiskSpace(ctx, &multinodepb.DiskSpaceRequest{
		Header: &multinodepb.RequestHeader{
			ApiKey: node.APISecret[:],
		},
	})
	if err != nil {
		return DiskSpace{}, Error.Wrap(err)
	}

	return DiskSpace{
		Allocated: diskSpaceResponse.Allocated,
		Used:      diskSpaceResponse.UsedPieces,
		Trash:     diskSpaceResponse.UsedTrash,
		Free:      diskSpaceResponse.Free,
		Available: diskSpaceResponse.Available,
		Overused:  diskSpaceResponse.Overused,
	}, nil
}

// dialUsage dials node and retrieves it's storage usage for provided interval.
func (service *Service) dialUsage(ctx context.Context, node nodes.Node, from, to time.Time) (_ Usage, err error) {
	defer mon.Task()(&ctx)(&err)

	conn, err := service.dialer.DialNodeURL(ctx, storx.NodeURL{
		ID:      node.ID,
		Address: node.PublicAddress,
	})
	if err != nil {
		return Usage{}, nodes.ErrNodeNotReachable.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, conn.Close())
	}()

	storageClient := multinodepb.NewDRPCStorageClient(conn)

	req := &multinodepb.StorageUsageRequest{
		Header: &multinodepb.RequestHeader{
			ApiKey: node.APISecret[:],
		},
		From: from,
		To:   to,
	}
	resp, err := storageClient.Usage(ctx, req)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	var stamps []UsageStamp
	for _, usage := range resp.GetStorageUsage() {
		stamps = append(stamps, UsageStamp{
			AtRestTotal:   usage.GetAtRestTotal(),
			IntervalStart: usage.GetIntervalStart(),
		})
	}

	return Usage{
		Stamps:  stamps,
		Summary: resp.GetSummary(),
	}, nil
}

// dialUsageSatellite dials node and retrieves it's storage usage for provided interval and satellite.
func (service *Service) dialUsageSatellite(ctx context.Context, node nodes.Node, satelliteID storx.NodeID, from, to time.Time) (_ Usage, err error) {
	defer mon.Task()(&ctx)(&err)

	conn, err := service.dialer.DialNodeURL(ctx, storx.NodeURL{
		ID:      node.ID,
		Address: node.PublicAddress,
	})
	if err != nil {
		return Usage{}, nodes.ErrNodeNotReachable.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, conn.Close())
	}()

	storageClient := multinodepb.NewDRPCStorageClient(conn)

	req := &multinodepb.StorageUsageSatelliteRequest{
		Header: &multinodepb.RequestHeader{
			ApiKey: node.APISecret[:],
		},
		SatelliteId: satelliteID,
		From:        from,
		To:          to,
	}
	resp, err := storageClient.UsageSatellite(ctx, req)
	if err != nil {
		return Usage{}, Error.Wrap(err)
	}

	var stamps []UsageStamp
	for _, usage := range resp.GetStorageUsage() {
		stamps = append(stamps, UsageStamp{
			AtRestTotal:   usage.GetAtRestTotal(),
			IntervalStart: usage.GetIntervalStart(),
		})
	}

	return Usage{
		Stamps:  stamps,
		Summary: resp.GetSummary(),
	}, nil
}