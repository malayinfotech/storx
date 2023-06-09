// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package testdata

import (
	"storx/storagenode/storagenodedb"
)

var v30 = MultiDBState{
	Version: 30,
	DBStates: DBStates{
		storagenodedb.UsedSerialsDBName:  v28.DBStates[storagenodedb.UsedSerialsDBName],
		storagenodedb.StorageUsageDBName: v28.DBStates[storagenodedb.StorageUsageDBName],
		storagenodedb.ReputationDBName:   v28.DBStates[storagenodedb.ReputationDBName],
		storagenodedb.PieceSpaceUsedDBName: &DBState{
			SQL: `
				CREATE TABLE piece_space_used (
					total INTEGER NOT NULL DEFAULT 0,
					content_size INTEGER NOT NULL,
					satellite_id BLOB
				);
				CREATE UNIQUE INDEX idx_piece_space_used_satellite_id ON piece_space_used(satellite_id);
				INSERT INTO piece_space_used (content_size, total) VALUES (1337, 1337);
				INSERT INTO piece_space_used (content_size, total, satellite_id) VALUES (1337, 1337, X'0ed28abb2813e184a1e98b0f6605c4911ea468c7e8433eb583e0fca7ceac3000');
			`,
			NewData: `
				INSERT INTO piece_space_used (content_size, total, satellite_id) VALUES (-5, -10, X'0ed28abb2813e184a1e98b0f6605c4911ea468c7e8433eb583e0fca7ceac3001');
			`,
		},
		storagenodedb.PieceInfoDBName:       v28.DBStates[storagenodedb.PieceInfoDBName],
		storagenodedb.PieceExpirationDBName: v28.DBStates[storagenodedb.PieceExpirationDBName],
		storagenodedb.OrdersDBName:          v28.DBStates[storagenodedb.OrdersDBName],
		storagenodedb.BandwidthDBName:       v28.DBStates[storagenodedb.BandwidthDBName],
		storagenodedb.SatellitesDBName:      v28.DBStates[storagenodedb.SatellitesDBName],
		storagenodedb.DeprecatedInfoDBName:  v28.DBStates[storagenodedb.DeprecatedInfoDBName],
		storagenodedb.NotificationsDBName:   v28.DBStates[storagenodedb.NotificationsDBName],
	},
}
