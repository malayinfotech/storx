// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

// Package inspector provides a private endpoint for monitoring status.
package inspector

import (
	"context"
	"net"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/rpc/rpcstatus"
	"storx/storagenode/bandwidth"
	"storx/storagenode/contact"
	"storx/storagenode/internalpb"
	"storx/storagenode/pieces"
	"storx/storagenode/piecestore"
)

var (
	mon = monkit.Package()

	// Error is the default error class for piecestore monitor errors.
	Error = errs.Class("piecestore inspector")
)

// Endpoint implements the inspector endpoints.
//
// architecture: Endpoint
type Endpoint struct {
	internalpb.DRPCPieceStoreInspectorUnimplementedServer

	log        *zap.Logger
	pieceStore *pieces.Store
	contact    *contact.Service
	pingStats  *contact.PingStats
	usageDB    bandwidth.DB

	startTime        time.Time
	pieceStoreConfig piecestore.OldConfig
	dashboardAddress net.Addr
	externalAddress  string
}

// NewEndpoint creates piecestore inspector instance.
func NewEndpoint(
	log *zap.Logger,
	pieceStore *pieces.Store,
	contact *contact.Service,
	pingStats *contact.PingStats,
	usageDB bandwidth.DB,
	pieceStoreConfig piecestore.OldConfig,
	dashboardAddress net.Addr,
	externalAddress string) *Endpoint {

	return &Endpoint{
		log:              log,
		pieceStore:       pieceStore,
		contact:          contact,
		pingStats:        pingStats,
		usageDB:          usageDB,
		pieceStoreConfig: pieceStoreConfig,
		dashboardAddress: dashboardAddress,
		startTime:        time.Now(),
		externalAddress:  externalAddress,
	}
}

// Stats returns current statistics about the storage node.
func (inspector *Endpoint) Stats(ctx context.Context, in *internalpb.StatsRequest) (out *internalpb.StatSummaryResponse, err error) {
	defer mon.Task()(&ctx)(&err)
	return inspector.retrieveStats(ctx)
}

func (inspector *Endpoint) retrieveStats(ctx context.Context) (_ *internalpb.StatSummaryResponse, err error) {
	defer mon.Task()(&ctx)(&err)

	// Space Usage
	piecesContentSize, err := inspector.pieceStore.SpaceUsedForPiecesAndTrash(ctx)
	if err != nil {
		return nil, rpcstatus.Error(rpcstatus.Internal, err.Error())
	}
	usage, err := bandwidth.TotalMonthlySummary(ctx, inspector.usageDB)
	if err != nil {
		return nil, rpcstatus.Error(rpcstatus.Internal, err.Error())
	}
	ingress := usage.Put + usage.PutRepair
	egress := usage.Get + usage.GetAudit + usage.GetRepair

	totalUsedBandwidth := usage.Total()
	availableSpace := inspector.pieceStoreConfig.AllocatedDiskSpace.Int64() - piecesContentSize

	return &internalpb.StatSummaryResponse{
		UsedSpace:      piecesContentSize,
		AvailableSpace: availableSpace,
		UsedIngress:    ingress,
		UsedEgress:     egress,
		UsedBandwidth:  totalUsedBandwidth,
	}, nil
}

// Dashboard returns dashboard information.
func (inspector *Endpoint) Dashboard(ctx context.Context, in *internalpb.DashboardRequest) (out *internalpb.DashboardResponse, err error) {
	defer mon.Task()(&ctx)(&err)
	return inspector.getDashboardData(ctx)
}

func (inspector *Endpoint) getDashboardData(ctx context.Context) (_ *internalpb.DashboardResponse, err error) {
	defer mon.Task()(&ctx)(&err)

	statsSummary, err := inspector.retrieveStats(ctx)
	if err != nil {
		return nil, rpcstatus.Error(rpcstatus.Internal, err.Error())
	}

	lastPingedAt := inspector.pingStats.WhenLastPinged()
	self := inspector.contact.Local()
	return &internalpb.DashboardResponse{
		NodeId:           self.ID,
		InternalAddress:  "",
		ExternalAddress:  self.Address,
		LastPinged:       lastPingedAt,
		DashboardAddress: inspector.dashboardAddress.String(),
		Uptime:           time.Since(inspector.startTime).String(),
		Stats:            statsSummary,
	}, nil
}
