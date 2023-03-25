// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package consoleapi

import (
	"net/http"

	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"common/uuid"
	"storx/private/web"
	"storx/satellite/console"
)

var (
	// ErrAPIKeysAPI - console api keys api error type.
	ErrAPIKeysAPI = errs.Class("console api keys")
)

// APIKeys is an api controller that exposes all api keys related functionality.
type APIKeys struct {
	log     *zap.Logger
	service *console.Service
}

// NewAPIKeys is a constructor for api api keys controller.
func NewAPIKeys(log *zap.Logger, service *console.Service) *APIKeys {
	return &APIKeys{
		log:     log,
		service: service,
	}
}

// DeleteByNameAndProjectID deletes specific api key by it's name and project ID.
// ID here may be project.publicID or project.ID.
func (keys *APIKeys) DeleteByNameAndProjectID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	defer mon.Task()(&ctx)(&err)

	name := r.URL.Query().Get("name")
	projectIDString := r.URL.Query().Get("projectID")
	publicIDString := r.URL.Query().Get("publicID")

	if name == "" {
		keys.serveJSONError(w, http.StatusBadRequest, err)
		return
	}

	var projectID uuid.UUID
	if projectIDString != "" {
		projectID, err = uuid.FromString(projectIDString)
		if err != nil {
			keys.serveJSONError(w, http.StatusBadRequest, err)
			return
		}
	} else if publicIDString != "" {
		projectID, err = uuid.FromString(publicIDString)
		if err != nil {
			keys.serveJSONError(w, http.StatusBadRequest, err)
			return
		}
	} else {
		keys.serveJSONError(w, http.StatusBadRequest, errs.New("Project ID was not provided."))
		return
	}

	err = keys.service.DeleteAPIKeyByNameAndProjectID(ctx, name, projectID)
	if err != nil {
		if console.ErrUnauthorized.Has(err) {
			keys.serveJSONError(w, http.StatusUnauthorized, err)
			return
		}

		if console.ErrNoAPIKey.Has(err) {
			keys.serveJSONError(w, http.StatusNoContent, err)
			return
		}

		keys.serveJSONError(w, http.StatusInternalServerError, err)
		return
	}
}

// serveJSONError writes JSON error to response output stream.
func (keys *APIKeys) serveJSONError(w http.ResponseWriter, status int, err error) {
	web.ServeJSONError(keys.log, w, status, err)
}
