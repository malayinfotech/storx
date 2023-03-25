// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package storagenodedb

import "private/tagsql"

// dbContainerImpl fulfills the migrate.DB interface and the SQLDB interface.
type dbContainerImpl struct {
	tagsql.DB
}

// Configure sets the underlining SQLDB connection.
func (db *dbContainerImpl) Configure(newDB tagsql.DB) {
	db.DB = newDB
}

// GetDB returns underlying implementation of dbContainerImpl.
func (db *dbContainerImpl) GetDB() tagsql.DB {
	return db.DB
}
