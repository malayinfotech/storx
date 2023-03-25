// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package audit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/spacemonkeygo/monkit/v3"
	"github.com/vivint/infectious"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/errs2"
	"common/identity"
	"common/memory"
	"common/pb"
	"common/rpc"
	"common/rpc/rpcpool"
	"common/rpc/rpcstatus"
	"common/storx"
	"storx/satellite/metabase"
	"storx/satellite/orders"
	"storx/satellite/overlay"
	"uplink/private/piecestore"
)

var (
	mon = monkit.Package()

	// ErrNotEnoughShares is the errs class for when not enough shares are available to do an audit.
	ErrNotEnoughShares = errs.Class("not enough shares for successful audit")
	// ErrSegmentDeleted is the errs class when the audited segment was deleted during the audit.
	ErrSegmentDeleted = errs.Class("segment deleted during audit")
	// ErrSegmentModified is the errs class used when a segment has been changed in any way.
	ErrSegmentModified = errs.Class("segment has been modified")
)

// Share represents required information about an audited share.
type Share struct {
	Error    error
	PieceNum int
	NodeID   storx.NodeID
	Data     []byte
}

// Verifier helps verify the correctness of a given stripe.
//
// architecture: Worker
type Verifier struct {
	log                *zap.Logger
	metabase           *metabase.DB
	orders             *orders.Service
	auditor            *identity.PeerIdentity
	dialer             rpc.Dialer
	overlay            *overlay.Service
	containment        Containment
	minBytesPerSecond  memory.Size
	minDownloadTimeout time.Duration

	nowFn                            func() time.Time
	OnTestingCheckSegmentAlteredHook func()
}

// NewVerifier creates a Verifier.
func NewVerifier(log *zap.Logger, metabase *metabase.DB, dialer rpc.Dialer, overlay *overlay.Service, containment Containment, orders *orders.Service, id *identity.FullIdentity, minBytesPerSecond memory.Size, minDownloadTimeout time.Duration) *Verifier {
	return &Verifier{
		log:                log,
		metabase:           metabase,
		orders:             orders,
		auditor:            id.PeerIdentity(),
		dialer:             dialer,
		overlay:            overlay,
		containment:        containment,
		minBytesPerSecond:  minBytesPerSecond,
		minDownloadTimeout: minDownloadTimeout,
		nowFn:              time.Now,
	}
}

// Verify downloads shares then verifies the data correctness at a random stripe.
func (verifier *Verifier) Verify(ctx context.Context, segment Segment, skip map[storx.NodeID]bool) (report Report, err error) {
	defer mon.Task()(&ctx)(&err)

	var segmentInfo metabase.Segment
	defer func() {
		recordStats(report, len(segmentInfo.Pieces), err)
	}()

	if segment.Expired(verifier.nowFn()) {
		verifier.log.Debug("segment expired before Verify")
		return Report{}, nil
	}

	segmentInfo, err = verifier.metabase.GetSegmentByPosition(ctx, metabase.GetSegmentByPosition{
		StreamID: segment.StreamID,
		Position: segment.Position,
	})
	if err != nil {
		if metabase.ErrSegmentNotFound.Has(err) {
			verifier.log.Debug("segment deleted before Verify")
			return Report{}, nil
		}
		return Report{}, err
	}

	randomIndex, err := GetRandomStripe(ctx, segmentInfo)
	if err != nil {
		return Report{}, err
	}

	var offlineNodes storx.NodeIDList
	var failedNodes storx.NodeIDList
	var unknownNodes storx.NodeIDList
	containedNodes := make(map[int]storx.NodeID)
	sharesToAudit := make(map[int]Share)

	orderLimits, privateKey, cachedNodesInfo, err := verifier.orders.CreateAuditOrderLimits(ctx, segmentInfo, skip)
	if err != nil {
		if orders.ErrDownloadFailedNotEnoughPieces.Has(err) {
			mon.Counter("not_enough_shares_for_audit").Inc(1)   //mon:locked
			mon.Counter("audit_not_enough_nodes_online").Inc(1) //mon:locked
			err = ErrNotEnoughShares.Wrap(err)
		}
		return Report{}, err
	}
	cachedNodesReputation := make(map[storx.NodeID]overlay.ReputationStatus, len(cachedNodesInfo))
	for id, info := range cachedNodesInfo {
		cachedNodesReputation[id] = info.Reputation
	}
	defer func() { report.NodesReputation = cachedNodesReputation }()

	// NOTE offlineNodes will include disqualified nodes because they aren't in
	// the skip list
	offlineNodes = getOfflineNodes(segmentInfo, orderLimits, skip)
	if len(offlineNodes) > 0 {
		verifier.log.Debug("Verify: order limits not created for some nodes (offline/disqualified)",
			zap.Strings("Node IDs", offlineNodes.Strings()),
			zap.String("Segment", segmentInfoString(segment)))
	}

	shares, err := verifier.DownloadShares(ctx, orderLimits, privateKey, cachedNodesInfo, randomIndex, segmentInfo.Redundancy.ShareSize)
	if err != nil {
		return Report{
			Offlines: offlineNodes,
		}, err
	}

	err = verifier.checkIfSegmentAltered(ctx, segmentInfo)
	if err != nil {
		if ErrSegmentDeleted.Has(err) {
			verifier.log.Debug("segment deleted during Verify")
			return Report{}, nil
		}
		if ErrSegmentModified.Has(err) {
			verifier.log.Debug("segment modified during Verify")
			return Report{}, nil
		}
		return Report{
			Offlines: offlineNodes,
		}, err
	}

	for pieceNum, share := range shares {
		if share.Error == nil {
			// no error -- share downloaded successfully
			sharesToAudit[pieceNum] = share
			continue
		}

		// TODO: just because an error came from the common/rpc package
		// does not decisively mean that the problem is something to do
		// with dialing. instead of trying to guess what different
		// error classes mean, we should make GetShare inside
		// DownloadShares return more direct information about when
		// the failure happened.
		if rpc.Error.Has(share.Error) {
			if errors.Is(share.Error, context.DeadlineExceeded) || errs.Is(share.Error, context.DeadlineExceeded) {
				// dial timeout
				offlineNodes = append(offlineNodes, share.NodeID)
				verifier.log.Debug("Verify: dial timeout (offline)",
					zap.Stringer("Node ID", share.NodeID),
					zap.String("Segment", segmentInfoString(segment)),
					zap.Error(share.Error))
				continue
			}
			if errs2.IsRPC(share.Error, rpcstatus.Unknown) {
				// dial failed -- offline node
				// TODO: we should never assume what an unknown
				// error means. This should be looking for an explicit
				// indication that dialing failed, not assuming dialing
				// failed because the rpc status is unknown
				offlineNodes = append(offlineNodes, share.NodeID)
				verifier.log.Debug("Verify: dial failed (offline)",
					zap.Stringer("Node ID", share.NodeID),
					zap.String("Segment", segmentInfoString(segment)),
					zap.Error(share.Error))
				continue
			}
			// unknown transport error
			unknownNodes = append(unknownNodes, share.NodeID)
			verifier.log.Info("Verify: unknown transport error (skipped)",
				zap.Stringer("Node ID", share.NodeID),
				zap.String("Segment", segmentInfoString(segment)),
				zap.Error(share.Error),
				zap.String("ErrorType", spew.Sprintf("%#+v", share.Error)))
			continue
		}

		if errs2.IsRPC(share.Error, rpcstatus.NotFound) {
			// missing share
			failedNodes = append(failedNodes, share.NodeID)
			verifier.log.Info("Verify: piece not found (audit failed)",
				zap.Stringer("Node ID", share.NodeID),
				zap.String("Segment", segmentInfoString(segment)),
				zap.Error(share.Error))
			continue
		}

		if errs2.IsRPC(share.Error, rpcstatus.DeadlineExceeded) {
			// dial successful, but download timed out
			containedNodes[pieceNum] = share.NodeID
			verifier.log.Info("Verify: download timeout (contained)",
				zap.Stringer("Node ID", share.NodeID),
				zap.String("Segment", segmentInfoString(segment)),
				zap.Error(share.Error))
			continue
		}

		// unknown error
		unknownNodes = append(unknownNodes, share.NodeID)
		verifier.log.Info("Verify: unknown error (skipped)",
			zap.Stringer("Node ID", share.NodeID),
			zap.String("Segment", segmentInfoString(segment)),
			zap.Error(share.Error),
			zap.String("ErrorType", spew.Sprintf("%#+v", share.Error)))
	}
	mon.IntVal("verify_shares_downloaded_successfully").Observe(int64(len(sharesToAudit))) //mon:locked

	required := segmentInfo.Redundancy.RequiredShares
	total := segmentInfo.Redundancy.TotalShares

	if len(sharesToAudit) < int(required) {
		mon.Counter("not_enough_shares_for_audit").Inc(1) //mon:locked
		// if we have reached this point, most likely something went wrong
		// like a network problem or a forgotten delete. Don't fail nodes.
		// We have an alert on this. Check the logs and see what happened.
		if len(offlineNodes)+len(containedNodes) > len(sharesToAudit)+len(failedNodes)+len(unknownNodes) {
			mon.Counter("audit_suspected_network_problem").Inc(1) //mon:locked
		} else {
			mon.Counter("audit_not_enough_shares_acquired").Inc(1) //mon:locked
		}
		report := Report{
			Offlines: offlineNodes,
			Unknown:  unknownNodes,
		}
		return report, ErrNotEnoughShares.New("got: %d, required: %d, failed: %d, offline: %d, unknown: %d, contained: %d",
			len(sharesToAudit), required, len(failedNodes), len(offlineNodes), len(unknownNodes), len(containedNodes))
	}
	// ensure we get values, even if only zero values, so that redash can have an alert based on these
	mon.Counter("not_enough_shares_for_audit").Inc(0)      //mon:locked
	mon.Counter("audit_not_enough_nodes_online").Inc(0)    //mon:locked
	mon.Counter("audit_not_enough_shares_acquired").Inc(0) //mon:locked
	mon.Counter("could_not_verify_audit_shares").Inc(0)    //mon:locked
	mon.Counter("audit_suspected_network_problem").Inc(0)  //mon:locked

	pieceNums, _, err := auditShares(ctx, required, total, sharesToAudit)
	if err != nil {
		mon.Counter("could_not_verify_audit_shares").Inc(1) //mon:locked
		verifier.log.Error("could not verify shares", zap.String("Segment", segmentInfoString(segment)), zap.Error(err))
		return Report{
			Fails:    failedNodes,
			Offlines: offlineNodes,
			Unknown:  unknownNodes,
		}, err
	}

	for _, pieceNum := range pieceNums {
		verifier.log.Info("Verify: share data altered (audit failed)",
			zap.Stringer("Node ID", shares[pieceNum].NodeID),
			zap.String("Segment", segmentInfoString(segment)))
		failedNodes = append(failedNodes, shares[pieceNum].NodeID)
	}

	successNodes := getSuccessNodes(ctx, shares, failedNodes, offlineNodes, unknownNodes, containedNodes)

	pendingAudits, err := createPendingAudits(ctx, containedNodes, segment)
	if err != nil {
		return Report{
			Successes: successNodes,
			Fails:     failedNodes,
			Offlines:  offlineNodes,
			Unknown:   unknownNodes,
		}, err
	}

	return Report{
		Successes:     successNodes,
		Fails:         failedNodes,
		Offlines:      offlineNodes,
		PendingAudits: pendingAudits,
		Unknown:       unknownNodes,
	}, nil
}

func segmentInfoString(segment Segment) string {
	return fmt.Sprintf("%s/%d",
		segment.StreamID.String(),
		segment.Position.Encode(),
	)
}

// DownloadShares downloads shares from the nodes where remote pieces are located.
func (verifier *Verifier) DownloadShares(ctx context.Context, limits []*pb.AddressedOrderLimit, piecePrivateKey storx.PiecePrivateKey, cachedNodesInfo map[storx.NodeID]overlay.NodeReputation, stripeIndex int32, shareSize int32) (shares map[int]Share, err error) {
	defer mon.Task()(&ctx)(&err)

	shares = make(map[int]Share, len(limits))
	ch := make(chan *Share, len(limits))

	for i, limit := range limits {
		if limit == nil {
			ch <- nil
			continue
		}

		var ipPort string
		node, ok := cachedNodesInfo[limit.Limit.StorageNodeId]
		if ok && node.LastIPPort != "" {
			ipPort = node.LastIPPort
		}

		go func(i int, limit *pb.AddressedOrderLimit) {
			share, err := verifier.GetShare(ctx, limit, piecePrivateKey, ipPort, stripeIndex, shareSize, i)
			if err != nil {
				share = Share{
					Error:    err,
					PieceNum: i,
					NodeID:   limit.GetLimit().StorageNodeId,
					Data:     nil,
				}
			}
			ch <- &share
		}(i, limit)
	}

	for range limits {
		share := <-ch
		if share != nil {
			shares[share.PieceNum] = *share
		}
	}

	return shares, nil
}

// IdentifyContainedNodes returns the set of all contained nodes out of the
// holders of pieces in the given segment.
func (verifier *Verifier) IdentifyContainedNodes(ctx context.Context, segment Segment) (skipList map[storx.NodeID]bool, err error) {
	segmentInfo, err := verifier.metabase.GetSegmentByPosition(ctx, metabase.GetSegmentByPosition{
		StreamID: segment.StreamID,
		Position: segment.Position,
	})
	if err != nil {
		return nil, err
	}

	skipList = make(map[storx.NodeID]bool)
	for _, piece := range segmentInfo.Pieces {
		_, err := verifier.containment.Get(ctx, piece.StorageNode)
		if err != nil {
			if ErrContainedNotFound.Has(err) {
				continue
			}
			verifier.log.Error("can not determine if node is contained", zap.Stringer("node-id", piece.StorageNode), zap.Error(err))
			continue
		}
		skipList[piece.StorageNode] = true
	}
	return skipList, nil
}

// GetShare use piece store client to download shares from nodes.
func (verifier *Verifier) GetShare(ctx context.Context, limit *pb.AddressedOrderLimit, piecePrivateKey storx.PiecePrivateKey, cachedIPAndPort string, stripeIndex, shareSize int32, pieceNum int) (share Share, err error) {
	defer mon.Task()(&ctx)(&err)

	bandwidthMsgSize := shareSize

	// determines number of seconds allotted for receiving data from a storage node
	timedCtx := ctx
	if verifier.minBytesPerSecond > 0 {
		maxTransferTime := time.Duration(int64(time.Second) * int64(bandwidthMsgSize) / verifier.minBytesPerSecond.Int64())
		if maxTransferTime < verifier.minDownloadTimeout {
			maxTransferTime = verifier.minDownloadTimeout
		}
		var cancel func()
		timedCtx, cancel = context.WithTimeout(ctx, maxTransferTime)
		defer cancel()
	}

	targetNodeID := limit.GetLimit().StorageNodeId
	log := verifier.log.Named(targetNodeID.String())
	var ps *piecestore.Client

	// if cached IP is given, try connecting there first
	if cachedIPAndPort != "" {
		nodeAddr := storx.NodeURL{
			ID:      targetNodeID,
			Address: cachedIPAndPort,
		}
		ps, err = piecestore.Dial(rpcpool.WithForceDial(timedCtx), verifier.dialer, nodeAddr, piecestore.DefaultConfig)
		if err != nil {
			log.Debug("failed to connect to audit target node at cached IP", zap.String("cached-ip-and-port", cachedIPAndPort), zap.Error(err))
		}
	}

	// if no cached IP was given, or connecting to cached IP failed, use node address
	if ps == nil {
		nodeAddr := storx.NodeURL{
			ID:      targetNodeID,
			Address: limit.GetStorageNodeAddress().Address,
		}
		ps, err = piecestore.Dial(rpcpool.WithForceDial(timedCtx), verifier.dialer, nodeAddr, piecestore.DefaultConfig)
		if err != nil {
			return Share{}, Error.Wrap(err)
		}
	}

	defer func() {
		err := ps.Close()
		if err != nil {
			verifier.log.Error("audit verifier failed to close conn to node: %+v", zap.Error(err))
		}
	}()

	offset := int64(shareSize) * int64(stripeIndex)

	downloader, err := ps.Download(timedCtx, limit.GetLimit(), piecePrivateKey, offset, int64(shareSize))
	if err != nil {
		return Share{}, err
	}
	defer func() { err = errs.Combine(err, downloader.Close()) }()

	buf := make([]byte, shareSize)
	_, err = io.ReadFull(downloader, buf)
	if err != nil {
		return Share{}, err
	}

	return Share{
		Error:    nil,
		PieceNum: pieceNum,
		NodeID:   targetNodeID,
		Data:     buf,
	}, nil
}

// checkIfSegmentAltered checks if oldSegment has been altered since it was selected for audit.
func (verifier *Verifier) checkIfSegmentAltered(ctx context.Context, oldSegment metabase.Segment) (err error) {
	defer mon.Task()(&ctx)(&err)

	if verifier.OnTestingCheckSegmentAlteredHook != nil {
		verifier.OnTestingCheckSegmentAlteredHook()
	}

	newSegment, err := verifier.metabase.GetSegmentByPosition(ctx, metabase.GetSegmentByPosition{
		StreamID: oldSegment.StreamID,
		Position: oldSegment.Position,
	})
	if err != nil {
		if metabase.ErrSegmentNotFound.Has(err) {
			return ErrSegmentDeleted.New("StreamID: %q Position: %d", oldSegment.StreamID.String(), oldSegment.Position.Encode())
		}
		return err
	}

	if !oldSegment.Pieces.Equal(newSegment.Pieces) {
		return ErrSegmentModified.New("StreamID: %q Position: %d", oldSegment.StreamID.String(), oldSegment.Position.Encode())
	}
	return nil
}

// SetNow allows tests to have the server act as if the current time is whatever they want.
func (verifier *Verifier) SetNow(nowFn func() time.Time) {
	verifier.nowFn = nowFn
}

// auditShares takes the downloaded shares and uses infectious's Correct function to check that they
// haven't been altered. auditShares returns a slice containing the piece numbers of altered shares,
// and a slice of the corrected shares.
func auditShares(ctx context.Context, required, total int16, originals map[int]Share) (pieceNums []int, corrected []infectious.Share, err error) {
	defer mon.Task()(&ctx)(&err)
	f, err := infectious.NewFEC(int(required), int(total))
	if err != nil {
		return nil, nil, err
	}

	copies, err := makeCopies(ctx, originals)
	if err != nil {
		return nil, nil, err
	}

	err = f.Correct(copies)
	if err != nil {
		return nil, nil, err
	}

	for _, share := range copies {
		if !bytes.Equal(originals[share.Number].Data, share.Data) {
			pieceNums = append(pieceNums, share.Number)
		}
	}
	return pieceNums, copies, nil
}

// makeCopies takes in a map of audit Shares and deep copies their data to a slice of infectious Shares.
func makeCopies(ctx context.Context, originals map[int]Share) (copies []infectious.Share, err error) {
	defer mon.Task()(&ctx)(&err)
	copies = make([]infectious.Share, 0, len(originals))
	for _, original := range originals {
		copies = append(copies, infectious.Share{
			Data:   append([]byte{}, original.Data...),
			Number: original.PieceNum})
	}
	return copies, nil
}

// getOfflineNodes returns those storage nodes from the segment which have no
// order limit nor are skipped.
func getOfflineNodes(segment metabase.Segment, limits []*pb.AddressedOrderLimit, skip map[storx.NodeID]bool) storx.NodeIDList {
	var offlines storx.NodeIDList

	nodesWithLimit := make(map[storx.NodeID]bool, len(limits))
	for _, limit := range limits {
		if limit != nil {
			nodesWithLimit[limit.GetLimit().StorageNodeId] = true
		}
	}

	for _, piece := range segment.Pieces {
		if !nodesWithLimit[piece.StorageNode] && !skip[piece.StorageNode] {
			offlines = append(offlines, piece.StorageNode)
		}
	}

	return offlines
}

// getSuccessNodes uses the failed nodes, offline nodes and contained nodes arrays to determine which nodes passed the audit.
func getSuccessNodes(ctx context.Context, shares map[int]Share, failedNodes, offlineNodes, unknownNodes storx.NodeIDList, containedNodes map[int]storx.NodeID) (successNodes storx.NodeIDList) {
	defer mon.Task()(&ctx)(nil)
	fails := make(map[storx.NodeID]bool)
	for _, fail := range failedNodes {
		fails[fail] = true
	}
	for _, offline := range offlineNodes {
		fails[offline] = true
	}
	for _, unknown := range unknownNodes {
		fails[unknown] = true
	}
	for _, contained := range containedNodes {
		fails[contained] = true
	}

	for _, share := range shares {
		if !fails[share.NodeID] {
			successNodes = append(successNodes, share.NodeID)
		}
	}

	return successNodes
}

func createPendingAudits(ctx context.Context, containedNodes map[int]storx.NodeID, segment Segment) (pending []*ReverificationJob, err error) {
	defer mon.Task()(&ctx)(&err)

	if len(containedNodes) == 0 {
		return nil, nil
	}

	pending = make([]*ReverificationJob, 0, len(containedNodes))
	for pieceNum, nodeID := range containedNodes {
		pending = append(pending, &ReverificationJob{
			Locator: PieceLocator{
				NodeID:   nodeID,
				StreamID: segment.StreamID,
				Position: segment.Position,
				PieceNum: pieceNum,
			},
		})
	}

	return pending, nil
}

// GetRandomStripe takes a segment and returns a random stripe index within that segment.
func GetRandomStripe(ctx context.Context, segment metabase.Segment) (index int32, err error) {
	defer mon.Task()(&ctx)(&err)

	// the last segment could be smaller than stripe size
	if segment.EncryptedSize < segment.Redundancy.StripeSize() {
		return 0, nil
	}

	var src cryptoSource
	rnd := rand.New(src)
	numStripes := segment.Redundancy.StripeCount(segment.EncryptedSize)
	randomStripeIndex := rnd.Int31n(numStripes)

	return randomStripeIndex, nil
}

func recordStats(report Report, totalPieces int, verifyErr error) {
	// If an audit was able to complete without auditing any nodes, that means
	// the segment has been altered.
	if verifyErr == nil && len(report.Successes) == 0 {
		return
	}

	numOffline := len(report.Offlines)
	numSuccessful := len(report.Successes)
	numFailed := len(report.Fails)
	numContained := len(report.PendingAudits)
	numUnknown := len(report.Unknown)

	totalAudited := numSuccessful + numFailed + numOffline + numContained
	auditedPercentage := float64(totalAudited) / float64(totalPieces)
	offlinePercentage := float64(0)
	successfulPercentage := float64(0)
	failedPercentage := float64(0)
	containedPercentage := float64(0)
	unknownPercentage := float64(0)
	if totalAudited > 0 {
		offlinePercentage = float64(numOffline) / float64(totalAudited)
		successfulPercentage = float64(numSuccessful) / float64(totalAudited)
		failedPercentage = float64(numFailed) / float64(totalAudited)
		containedPercentage = float64(numContained) / float64(totalAudited)
		unknownPercentage = float64(numUnknown) / float64(totalAudited)
	}

	mon.Meter("audit_success_nodes_global").Mark(numSuccessful)     //mon:locked
	mon.Meter("audit_fail_nodes_global").Mark(numFailed)            //mon:locked
	mon.Meter("audit_offline_nodes_global").Mark(numOffline)        //mon:locked
	mon.Meter("audit_contained_nodes_global").Mark(numContained)    //mon:locked
	mon.Meter("audit_unknown_nodes_global").Mark(numUnknown)        //mon:locked
	mon.Meter("audit_total_nodes_global").Mark(totalAudited)        //mon:locked
	mon.Meter("audit_total_pointer_nodes_global").Mark(totalPieces) //mon:locked

	mon.IntVal("audit_success_nodes").Observe(int64(numSuccessful))           //mon:locked
	mon.IntVal("audit_fail_nodes").Observe(int64(numFailed))                  //mon:locked
	mon.IntVal("audit_offline_nodes").Observe(int64(numOffline))              //mon:locked
	mon.IntVal("audit_contained_nodes").Observe(int64(numContained))          //mon:locked
	mon.IntVal("audit_unknown_nodes").Observe(int64(numUnknown))              //mon:locked
	mon.IntVal("audit_total_nodes").Observe(int64(totalAudited))              //mon:locked
	mon.IntVal("audit_total_pointer_nodes").Observe(int64(totalPieces))       //mon:locked
	mon.FloatVal("audited_percentage").Observe(auditedPercentage)             //mon:locked
	mon.FloatVal("audit_offline_percentage").Observe(offlinePercentage)       //mon:locked
	mon.FloatVal("audit_successful_percentage").Observe(successfulPercentage) //mon:locked
	mon.FloatVal("audit_failed_percentage").Observe(failedPercentage)         //mon:locked
	mon.FloatVal("audit_contained_percentage").Observe(containedPercentage)   //mon:locked
	mon.FloatVal("audit_unknown_percentage").Observe(unknownPercentage)       //mon:locked
}
