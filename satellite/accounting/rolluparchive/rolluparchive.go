// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package rolluparchive

import (
	"context"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/sync2"
	"storx/satellite/accounting"
)

// Error is a standard error class for this package.
var (
	Error = errs.Class("rolluparchive")
	mon   = monkit.Package()
)

// Config contains configurable values for rollup archiver.
type Config struct {
	Interval   time.Duration `help:"how frequently rollup archiver should run" releaseDefault:"24h" devDefault:"120s" testDefault:"$TESTINTERVAL"`
	ArchiveAge time.Duration `help:"age at which a rollup is archived" default:"2160h" testDefault:"24h"`
	BatchSize  int           `help:"number of records to delete per delete execution. Used only for crdb which is slow without limit." default:"500" testDefault:"1000"`
	Enabled    bool          `help:"whether or not the rollup archive is enabled." default:"true"`
}

// Chore archives bucket and storagenode rollups at a given interval.
//
// architecture: Chore
type Chore struct {
	log               *zap.Logger
	Loop              *sync2.Cycle
	archiveAge        time.Duration
	batchSize         int
	nodeAccounting    accounting.StoragenodeAccounting
	projectAccounting accounting.ProjectAccounting
}

// New creates a new rollup archiver chore.
func New(log *zap.Logger, sdb accounting.StoragenodeAccounting, pdb accounting.ProjectAccounting, config Config) *Chore {
	return &Chore{
		log:               log,
		Loop:              sync2.NewCycle(config.Interval),
		archiveAge:        config.ArchiveAge,
		batchSize:         config.BatchSize,
		nodeAccounting:    sdb,
		projectAccounting: pdb,
	}
}

// Run starts the archiver chore.
func (chore *Chore) Run(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	if chore.archiveAge < 0 {
		return Error.New("archive age can't be less than 0")
	}
	return chore.Loop.Run(ctx, func(ctx context.Context) error {
		cutoff := time.Now().UTC().Add(-chore.archiveAge)
		err := chore.ArchiveRollups(ctx, cutoff, chore.batchSize)
		if err != nil {
			chore.log.Error("error archiving SN and bucket bandwidth rollups", zap.Error(err))
		}
		return nil
	})
}

// Close stops the service and releases any resources.
func (chore *Chore) Close() error {
	chore.Loop.Close()
	return nil
}

// ArchiveRollups will remove old rollups from active rollup tables.
func (chore *Chore) ArchiveRollups(ctx context.Context, cutoff time.Time, batchSize int) (err error) {
	defer mon.Task()(&ctx)(&err)
	nodeRollupsArchived, err := chore.nodeAccounting.ArchiveRollupsBefore(ctx, cutoff, batchSize)
	if err != nil {
		chore.log.Error("archiving bandwidth rollups", zap.Int("node rollups archived", nodeRollupsArchived), zap.Error(err))
		return Error.Wrap(err)
	}
	bucketRollupsArchived, err := chore.projectAccounting.ArchiveRollupsBefore(ctx, cutoff, batchSize)
	if err != nil {
		chore.log.Error("archiving bandwidth rollups", zap.Int("bucket rollups archived", bucketRollupsArchived), zap.Error(err))
		return Error.Wrap(err)
	}
	return nil
}
