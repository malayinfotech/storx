// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package audit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"common/storx"
	"common/testcontext"
	"storx/private/testplanet"
	"storx/satellite"
	"storx/satellite/audit"
	"storx/satellite/overlay"
)

func TestReportPendingAudits(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		audits := satellite.Audit
		audits.Worker.Loop.Pause()

		nodeID := planet.StorageNodes[0].ID()

		pending := audit.ReverificationJob{
			Locator: audit.PieceLocator{
				NodeID: nodeID,
			},
		}

		report := audit.Report{PendingAudits: []*audit.ReverificationJob{&pending}}
		containment := satellite.DB.Containment()

		audits.Reporter.RecordAudits(ctx, report)

		pa, err := containment.Get(ctx, nodeID)
		require.NoError(t, err)
		assert.Equal(t, pending.Locator, pa.Locator)
	})
}

func TestRecordAuditsAtLeastOnce(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				// disable reputation write cache so changes are immediate
				config.Reputation.FlushInterval = 0
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		audits := satellite.Audit
		audits.Worker.Loop.Pause()

		nodeID := planet.StorageNodes[0].ID()

		report := audit.Report{Successes: []storx.NodeID{nodeID}}

		// expect RecordAudits to try recording at least once (maxRetries is set to 0)
		audits.Reporter.RecordAudits(ctx, report)

		service := satellite.Reputation.Service
		node, err := service.Get(ctx, nodeID)
		require.NoError(t, err)
		require.EqualValues(t, 1, node.TotalAuditCount)
	})
}

// TestRecordAuditsCorrectOutcome ensures that audit successes, failures, and unknown audits result in the correct disqualification/suspension state.
func TestRecordAuditsCorrectOutcome(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 5, UplinkCount: 0,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				config.Reputation.InitialAlpha = 1
				config.Reputation.AuditLambda = 0.95
				config.Reputation.AuditDQ = 0.6
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		audits := satellite.Audit
		audits.Worker.Loop.Pause()

		goodNode := planet.StorageNodes[0].ID()
		dqNode := planet.StorageNodes[1].ID()
		suspendedNode := planet.StorageNodes[2].ID()
		pendingNode := planet.StorageNodes[3].ID()
		offlineNode := planet.StorageNodes[4].ID()

		report := audit.Report{
			Successes: []storx.NodeID{goodNode},
			Fails:     []storx.NodeID{dqNode},
			Unknown:   []storx.NodeID{suspendedNode},
			PendingAudits: []*audit.ReverificationJob{
				{
					Locator:       audit.PieceLocator{NodeID: pendingNode},
					ReverifyCount: 0,
				},
			},
			Offlines: []storx.NodeID{offlineNode},
		}

		audits.Reporter.RecordAudits(ctx, report)

		overlay := satellite.Overlay.Service
		node, err := overlay.Get(ctx, goodNode)
		require.NoError(t, err)
		require.Nil(t, node.Disqualified)
		require.Nil(t, node.UnknownAuditSuspended)

		node, err = overlay.Get(ctx, dqNode)
		require.NoError(t, err)
		require.NotNil(t, node.Disqualified)
		require.Nil(t, node.UnknownAuditSuspended)

		node, err = overlay.Get(ctx, suspendedNode)
		require.NoError(t, err)
		require.Nil(t, node.Disqualified)
		require.NotNil(t, node.UnknownAuditSuspended)

		node, err = overlay.Get(ctx, pendingNode)
		require.NoError(t, err)
		require.Nil(t, node.Disqualified)
		require.Nil(t, node.UnknownAuditSuspended)

		node, err = overlay.Get(ctx, offlineNode)
		require.NoError(t, err)
		require.Nil(t, node.Disqualified)
		require.Nil(t, node.UnknownAuditSuspended)
	})
}

func TestSuspensionTimeNotResetBySuccessiveAudit(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		audits := satellite.Audit
		audits.Worker.Loop.Pause()

		suspendedNode := planet.StorageNodes[0].ID()

		audits.Reporter.RecordAudits(ctx, audit.Report{Unknown: []storx.NodeID{suspendedNode}})

		overlay := satellite.Overlay.Service

		node, err := overlay.Get(ctx, suspendedNode)
		require.NoError(t, err)
		require.Nil(t, node.Disqualified)
		require.NotNil(t, node.UnknownAuditSuspended)

		suspendedAt := node.UnknownAuditSuspended

		audits.Reporter.RecordAudits(ctx, audit.Report{Unknown: []storx.NodeID{suspendedNode}})

		node, err = overlay.Get(ctx, suspendedNode)
		require.NoError(t, err)
		require.Nil(t, node.Disqualified)
		require.NotNil(t, node.UnknownAuditSuspended)
		require.Equal(t, suspendedAt, node.UnknownAuditSuspended)
	})
}

// TestGracefullyExitedNotUpdated verifies that a gracefully exited node's reputation, suspension,
// and disqualification flags are not updated when an audit is reported for that node.
func TestGracefullyExitedNotUpdated(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 5, UplinkCount: 0,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		audits := satellite.Audit
		audits.Worker.Loop.Pause()
		cache := satellite.Overlay.DB
		reputationDB := satellite.DB.Reputation()

		successNode := planet.StorageNodes[0]
		failedNode := planet.StorageNodes[1]
		containedNode := planet.StorageNodes[2]
		unknownNode := planet.StorageNodes[3]
		offlineNode := planet.StorageNodes[4]
		nodeList := []*testplanet.StorageNode{successNode, failedNode, containedNode, unknownNode, offlineNode}

		report := audit.Report{
			Successes: storx.NodeIDList{successNode.ID(), failedNode.ID(), containedNode.ID(), unknownNode.ID(), offlineNode.ID()},
		}
		audits.Reporter.RecordAudits(ctx, report)

		// mark each node as having gracefully exited
		for _, node := range nodeList {
			req := &overlay.ExitStatusRequest{
				NodeID:              node.ID(),
				ExitInitiatedAt:     time.Now(),
				ExitLoopCompletedAt: time.Now(),
				ExitFinishedAt:      time.Now(),
			}
			_, err := cache.UpdateExitStatus(ctx, req)
			require.NoError(t, err)
		}

		pending := audit.ReverificationJob{
			Locator: audit.PieceLocator{
				NodeID: containedNode.ID(),
			},
		}
		report = audit.Report{
			Successes:     storx.NodeIDList{successNode.ID()},
			Fails:         storx.NodeIDList{failedNode.ID()},
			Offlines:      storx.NodeIDList{offlineNode.ID()},
			PendingAudits: []*audit.ReverificationJob{&pending},
			Unknown:       storx.NodeIDList{unknownNode.ID()},
		}
		audits.Reporter.RecordAudits(ctx, report)

		// since every node has gracefully exit, reputation, dq, and suspension should remain at default values
		for _, node := range nodeList {
			nodeCacheInfo, err := reputationDB.Get(ctx, node.ID())
			require.NoError(t, err)

			require.Nil(t, nodeCacheInfo.UnknownAuditSuspended)
			require.Nil(t, nodeCacheInfo.Disqualified)
		}
	})
}

func TestReportOfflineAudits(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
		Reconfigure: testplanet.Reconfigure{
			Satellite: func(log *zap.Logger, index int, config *satellite.Config) {
				// disable reputation write cache so changes are immediate
				config.Reputation.FlushInterval = 0
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		satellite := planet.Satellites[0]
		node := planet.StorageNodes[0]
		audits := satellite.Audit
		audits.Worker.Loop.Pause()
		reputationService := satellite.Core.Reputation.Service

		audits.Reporter.RecordAudits(ctx, audit.Report{Offlines: storx.NodeIDList{node.ID()}})

		info, err := reputationService.Get(ctx, node.ID())
		require.NoError(t, err)
		require.Equal(t, int64(1), info.TotalAuditCount)

		// check that other reputation stats were not incorrectly updated by offline audit
		require.EqualValues(t, 0, info.AuditSuccessCount)
		require.EqualValues(t, satellite.Config.Reputation.InitialAlpha, info.AuditReputationAlpha)
		require.EqualValues(t, satellite.Config.Reputation.InitialBeta, info.AuditReputationBeta)
		require.EqualValues(t, 1, info.UnknownAuditReputationAlpha)
		require.EqualValues(t, 0, info.UnknownAuditReputationBeta)
	})
}
