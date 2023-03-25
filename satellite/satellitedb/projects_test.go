// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"common/testcontext"
	"common/uuid"
	"storx/satellite"
	"storx/satellite/console"
	"storx/satellite/satellitedb/satellitedbtest"
)

func TestProjectsGetByPublicID(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		projects := db.Console().Projects()

		prj, err := projects.Insert(ctx, &console.Project{
			Name:        "ProjectName",
			Description: "projects description",
		})
		require.NoError(t, err)
		require.NotNil(t, prj)

		pubID := prj.PublicID
		require.NotNil(t, pubID)
		require.False(t, pubID.IsZero())

		prj, err = projects.GetByPublicID(ctx, pubID)
		require.NoError(t, err)
		require.Equal(t, pubID, prj.PublicID)
	})
}

func TestProjectsGetSalt(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		projects := db.Console().Projects()

		prj, err := projects.Insert(ctx, &console.Project{
			Name:        "ProjectName",
			Description: "projects description",
		})
		require.NoError(t, err)
		require.NotNil(t, prj)

		salt, err := projects.GetSalt(ctx, prj.ID)
		require.NoError(t, err)

		_, err = uuid.FromBytes(salt)
		require.NoError(t, err)
	})
}

func TestUpdateProjectUsageLimits(t *testing.T) {
	satellitedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db satellite.DB) {
		limits := console.UsageLimits{Storage: rand.Int63(), Bandwidth: rand.Int63(), Segment: rand.Int63()}
		projectsRepo := db.Console().Projects()

		proj, err := projectsRepo.Insert(ctx, &console.Project{})
		require.NoError(t, err)
		require.NotNil(t, proj)

		err = projectsRepo.UpdateUsageLimits(ctx, proj.ID, limits)
		require.NoError(t, err)

		proj, err = projectsRepo.Get(ctx, proj.ID)
		require.NoError(t, err)
		require.Equal(t, limits.Bandwidth, proj.BandwidthLimit.Int64())
		require.Equal(t, limits.Storage, proj.StorageLimit.Int64())
		require.Equal(t, limits.Segment, *proj.SegmentLimit)
	})
}
