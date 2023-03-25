// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package apikeys_test

import (
	"testing"
	"time"

	"github.com/zeebo/assert"

	"common/testcontext"
	"storx/private/multinodeauth"
	"storx/storagenode"
	"storx/storagenode/apikeys"
	"storx/storagenode/storagenodedb/storagenodedbtest"
)

func TestAPIKeysDB(t *testing.T) {
	storagenodedbtest.Run(t, func(ctx *testcontext.Context, t *testing.T, db storagenode.DB) {
		apiKeys := db.APIKeys()
		secret, err := multinodeauth.NewSecret()
		assert.NoError(t, err)
		secret2, err := multinodeauth.NewSecret()
		assert.NoError(t, err)

		t.Run("Store", func(t *testing.T) {
			err := apiKeys.Store(ctx, apikeys.APIKey{
				Secret:    secret,
				CreatedAt: time.Now().UTC(),
			})
			assert.NoError(t, err)
		})

		t.Run("Check", func(t *testing.T) {
			err := apiKeys.Check(ctx, secret)
			assert.NoError(t, err)

			err = apiKeys.Check(ctx, secret2)
			assert.Error(t, err)
		})

		t.Run("Revoke", func(t *testing.T) {
			err = apiKeys.Revoke(ctx, secret)
			assert.NoError(t, err)

			err = apiKeys.Check(ctx, secret)
			assert.Error(t, err)
		})
	})
}
