// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package audit

import (
	"context"
	"strings"

	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/storx"
	"storx/satellite/overlay"
	"storx/satellite/reputation"
)

// reporter records audit reports in overlay and implements the Reporter interface.
//
// architecture: Service
type reporter struct {
	log              *zap.Logger
	reputations      *reputation.Service
	overlay          *overlay.Service
	containment      Containment
	maxRetries       int
	maxReverifyCount int32
}

// Reporter records audit reports in the overlay and database.
type Reporter interface {
	RecordAudits(ctx context.Context, req Report)
	ReportReverificationNeeded(ctx context.Context, piece *PieceLocator) (err error)
	RecordReverificationResult(ctx context.Context, pendingJob *ReverificationJob, outcome Outcome, reputation overlay.ReputationStatus) (err error)
}

// Report contains audit result.
// It records whether an audit is able to be completed, the total number of
// pieces a given audit has conducted for, lists for nodes that
// succeeded, failed, were offline, have pending audits, or failed for unknown
// reasons and their current reputation status.
type Report struct {
	Successes       storx.NodeIDList
	Fails           storx.NodeIDList
	Offlines        storx.NodeIDList
	PendingAudits   []*ReverificationJob
	Unknown         storx.NodeIDList
	NodesReputation map[storx.NodeID]overlay.ReputationStatus
}

// NewReporter instantiates a reporter.
func NewReporter(log *zap.Logger, reputations *reputation.Service, overlay *overlay.Service, containment Containment, maxRetries int, maxReverifyCount int32) Reporter {
	return &reporter{
		log:              log,
		reputations:      reputations,
		overlay:          overlay,
		containment:      containment,
		maxRetries:       maxRetries,
		maxReverifyCount: maxReverifyCount,
	}
}

// RecordAudits saves audit results, applying reputation changes as appropriate.
// If some records can not be updated after a number of attempts, the failures
// are logged at level ERROR, but are otherwise thrown away.
func (reporter *reporter) RecordAudits(ctx context.Context, req Report) {
	defer mon.Task()(&ctx)(nil)

	successes := req.Successes
	fails := req.Fails
	unknowns := req.Unknown
	offlines := req.Offlines
	pendingAudits := req.PendingAudits

	reporter.log.Debug("Reporting audits",
		zap.Int("successes", len(successes)),
		zap.Int("failures", len(fails)),
		zap.Int("unknowns", len(unknowns)),
		zap.Int("offlines", len(offlines)),
		zap.Int("pending", len(pendingAudits)),
	)

	nodesReputation := req.NodesReputation

	reportFailures := func(tries int, resultType string, err error, nodes storx.NodeIDList, pending []*ReverificationJob) {
		if err == nil || tries < reporter.maxRetries {
			// don't need to report anything until the last time through
			return
		}
		reporter.log.Error("failed to update reputation information with audit results",
			zap.String("result type", resultType),
			zap.Error(err),
			zap.String("node IDs", strings.Join(nodes.Strings(), ", ")),
			zap.Any("pending segment audits", pending))
	}

	var err error
	for tries := 0; tries <= reporter.maxRetries; tries++ {
		if len(successes) == 0 && len(fails) == 0 && len(unknowns) == 0 && len(offlines) == 0 && len(pendingAudits) == 0 {
			return
		}

		successes, err = reporter.recordAuditStatus(ctx, successes, nodesReputation, reputation.AuditSuccess)
		reportFailures(tries, "successful", err, successes, nil)
		fails, err = reporter.recordAuditStatus(ctx, fails, nodesReputation, reputation.AuditFailure)
		reportFailures(tries, "failed", err, fails, nil)
		unknowns, err = reporter.recordAuditStatus(ctx, unknowns, nodesReputation, reputation.AuditUnknown)
		reportFailures(tries, "unknown", err, unknowns, nil)
		offlines, err = reporter.recordAuditStatus(ctx, offlines, nodesReputation, reputation.AuditOffline)
		reportFailures(tries, "offline", err, offlines, nil)
		pendingAudits, err = reporter.recordPendingAudits(ctx, pendingAudits, nodesReputation)
		reportFailures(tries, "pending", err, nil, pendingAudits)
	}
}

func (reporter *reporter) recordAuditStatus(ctx context.Context, nodeIDs storx.NodeIDList, nodesReputation map[storx.NodeID]overlay.ReputationStatus, auditOutcome reputation.AuditType) (failed storx.NodeIDList, err error) {
	defer mon.Task()(&ctx)(&err)

	if len(nodeIDs) == 0 {
		return nil, nil
	}
	var errors errs.Group
	for _, nodeID := range nodeIDs {
		err = reporter.reputations.ApplyAudit(ctx, nodeID, nodesReputation[nodeID], auditOutcome)
		if err != nil {
			failed = append(failed, nodeID)
			errors.Add(Error.New("failed to record audit status %s in overlay for node %s: %w", auditOutcome.String(), nodeID.String(), err))
		}
	}
	return failed, errors.Err()
}

// recordPendingAudits updates the containment status of nodes with pending piece audits.
func (reporter *reporter) recordPendingAudits(ctx context.Context, pendingAudits []*ReverificationJob, nodesReputation map[storx.NodeID]overlay.ReputationStatus) (failed []*ReverificationJob, err error) {
	defer mon.Task()(&ctx)(&err)
	var errlist errs.Group

	for _, pendingAudit := range pendingAudits {
		logger := reporter.log.With(
			zap.Stringer("Node ID", pendingAudit.Locator.NodeID),
			zap.Stringer("Stream ID", pendingAudit.Locator.StreamID),
			zap.Uint64("Position", pendingAudit.Locator.Position.Encode()),
			zap.Int("Piece Num", pendingAudit.Locator.PieceNum))

		if pendingAudit.ReverifyCount < int(reporter.maxReverifyCount) {
			err := reporter.ReportReverificationNeeded(ctx, &pendingAudit.Locator)
			if err != nil {
				failed = append(failed, pendingAudit)
				errlist.Add(err)
				continue
			}
			logger.Info("reverification queued")
			continue
		}
		// record failure -- max reverify count reached
		logger.Info("max reverify count reached (audit failed)")
		err = reporter.reputations.ApplyAudit(ctx, pendingAudit.Locator.NodeID, nodesReputation[pendingAudit.Locator.NodeID], reputation.AuditFailure)
		if err != nil {
			logger.Info("failed to update reputation information", zap.Error(err))
			errlist.Add(err)
			failed = append(failed, pendingAudit)
			continue
		}
		_, stillContained, err := reporter.containment.Delete(ctx, &pendingAudit.Locator)
		if err != nil {
			if !ErrContainedNotFound.Has(err) {
				errlist.Add(err)
			}
			continue
		}
		if !stillContained {
			err = reporter.overlay.SetNodeContained(ctx, pendingAudit.Locator.NodeID, false)
			if err != nil {
				logger.Error("failed to mark node as not contained", zap.Error(err))
			}
		}
	}

	if len(failed) > 0 {
		return failed, errs.Combine(Error.New("failed to record some pending audits"), errlist.Err())
	}
	return nil, nil
}

func (reporter *reporter) ReportReverificationNeeded(ctx context.Context, piece *PieceLocator) (err error) {
	defer mon.Task()(&ctx)(&err)

	err = reporter.containment.Insert(ctx, piece)
	if err != nil {
		return Error.New("failed to queue reverification audit for node: %w", err)
	}

	err = reporter.overlay.SetNodeContained(ctx, piece.NodeID, true)
	if err != nil {
		return Error.New("failed to update contained status: %w", err)
	}
	return nil
}

func (reporter *reporter) RecordReverificationResult(ctx context.Context, pendingJob *ReverificationJob, outcome Outcome, reputation overlay.ReputationStatus) (err error) {
	defer mon.Task()(&ctx)(&err)

	keepInQueue := true
	report := Report{
		NodesReputation: map[storx.NodeID]overlay.ReputationStatus{
			pendingJob.Locator.NodeID: reputation,
		},
	}
	switch outcome {
	case OutcomeNotPerformed:
	case OutcomeNotNecessary:
		keepInQueue = false
	case OutcomeSuccess:
		report.Successes = append(report.Successes, pendingJob.Locator.NodeID)
		keepInQueue = false
	case OutcomeFailure:
		report.Fails = append(report.Fails, pendingJob.Locator.NodeID)
		keepInQueue = false
	case OutcomeTimedOut:
		// This will get re-added to the reverification queue, but that is idempotent
		// and fine. We do need to add it to PendingAudits in order to get the
		// maxReverifyCount check.
		report.PendingAudits = append(report.PendingAudits, pendingJob)
	case OutcomeUnknownError:
		report.Unknown = append(report.Unknown, pendingJob.Locator.NodeID)
		keepInQueue = false
	case OutcomeNodeOffline:
		report.Offlines = append(report.Offlines, pendingJob.Locator.NodeID)
	}
	var errList errs.Group

	// apply any necessary reputation changes
	reporter.RecordAudits(ctx, report)

	// remove from reverifications queue if appropriate
	if !keepInQueue {
		_, stillContained, err := reporter.containment.Delete(ctx, &pendingJob.Locator)
		if err != nil {
			if !ErrContainedNotFound.Has(err) {
				errList.Add(err)
			}
		} else if !stillContained {
			err = reporter.overlay.SetNodeContained(ctx, pendingJob.Locator.NodeID, false)
			errList.Add(err)
		}
	}
	return errList.Err()
}
