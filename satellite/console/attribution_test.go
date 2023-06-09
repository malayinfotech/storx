// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package console_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"common/storx"
	"common/testcontext"
	"common/testrand"
	"storx/private/testplanet"
	"storx/satellite/console"
)

func TestUsers(t *testing.T) {
	testplanet.Run(t, testplanet.Config{SatelliteCount: 1}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		sat := planet.Satellites[0]
		buckets := sat.API.Buckets.Service
		db := sat.DB
		consoleDB := db.Console()

		// create user
		userPassHash := testrand.Bytes(8)

		// create an user with no partnerID
		_, err := consoleDB.Users().Insert(ctx, &console.User{
			ID:           testrand.UUID(),
			FullName:     "John Doe",
			Email:        "john@mail.test",
			PasswordHash: userPassHash,
			Status:       console.Active,
		})
		require.NoError(t, err)

		// create a project with partnerID
		_, err = consoleDB.Projects().Insert(ctx, &console.Project{
			ID:          testrand.UUID(),
			Name:        "John Doe",
			Description: "some description",
			CreatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// create a project with no partnerID
		proj, err := consoleDB.Projects().Insert(ctx, &console.Project{
			ID:          testrand.UUID(),
			Name:        "John Doe",
			Description: "some description",
			CreatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// create a APIKey with no partnerID
		_, err = consoleDB.APIKeys().Create(ctx, testrand.Bytes(8), console.APIKeyInfo{
			ID:        testrand.UUID(),
			ProjectID: proj.ID,
			Name:      "John Doe",
			Secret:    []byte("xyz"),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		// create a bucket with no partnerID
		_, err = buckets.CreateBucket(ctx, storx.Bucket{
			ID:                  testrand.UUID(),
			Name:                "testbucket",
			ProjectID:           proj.ID,
			Created:             time.Now(),
			PathCipher:          storx.EncAESGCM,
			DefaultSegmentsSize: int64(100),
		})
		require.NoError(t, err)

		// update a bucket with partnerID
		bucket, err := buckets.UpdateBucket(ctx, storx.Bucket{
			ID:                  testrand.UUID(),
			Name:                "testbucket",
			ProjectID:           proj.ID,
			PartnerID:           proj.ID,
			Created:             time.Now(),
			PathCipher:          storx.EncAESGCM,
			DefaultSegmentsSize: int64(100),
		})
		require.NoError(t, err)
		require.Equal(t, proj.ID, bucket.PartnerID)

		bucket, err = buckets.GetBucket(ctx, []byte("testbucket"), proj.ID)
		require.NoError(t, err)
		require.Equal(t, proj.ID, bucket.PartnerID)
	})
}
