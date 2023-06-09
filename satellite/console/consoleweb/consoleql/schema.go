// Copyright (C) 2018 Storx Labs, Inc.
// See LICENSE for copying information.

package consoleql

import (
	"github.com/graphql-go/graphql"
	"go.uber.org/zap"

	"storx/satellite/console"
	"storx/satellite/mailservice"
)

// CreateSchema creates a schema for satellites console graphql api.
func CreateSchema(log *zap.Logger, service *console.Service, mailService *mailservice.Service) (schema graphql.Schema, err error) {
	creator := TypeCreator{}

	err = creator.Create(log, service, mailService)
	if err != nil {
		return
	}

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    creator.RootQuery(),
		Mutation: creator.RootMutation(),
	})
}
