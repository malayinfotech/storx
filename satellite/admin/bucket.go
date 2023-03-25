// Copyright (C) 2021 Storx Labs, Inc.
// See LICENSE for copying information.

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"common/storx"
	"common/uuid"
	"storx/satellite/buckets"
)

func validateBucketPathParameters(vars map[string]string) (project uuid.NullUUID, bucket []byte, err error) {
	projectUUIDString, ok := vars["project"]
	if !ok {
		return project, bucket, fmt.Errorf("project-uuid missing")
	}

	project.UUID, err = uuid.FromString(projectUUIDString)
	if err != nil {
		return project, bucket, fmt.Errorf("project-uuid is not a valid uuid")
	}
	project.Valid = true

	bucketName := vars["bucket"]
	if len(bucketName) == 0 {
		return project, bucket, fmt.Errorf("bucket name is missing")
	}

	bucket = []byte(bucketName)
	return
}

func parsePlacementConstraint(regionCode string) (storx.PlacementConstraint, error) {
	switch regionCode {
	case "EU":
		return storx.EU, nil
	case "EEA":
		return storx.EEA, nil
	case "US":
		return storx.US, nil
	case "DE":
		return storx.DE, nil
	case "":
		return storx.EveryCountry, fmt.Errorf("missing region parameter")
	default:
		return storx.EveryCountry, fmt.Errorf("unrecognized region parameter: %s", regionCode)
	}
}

func (server *Server) updateBucket(w http.ResponseWriter, r *http.Request, placement storx.PlacementConstraint) {
	ctx := r.Context()

	project, bucket, err := validateBucketPathParameters(mux.Vars(r))
	if err != nil {
		sendJSONError(w, err.Error(), "", http.StatusBadRequest)
		return
	}

	b, err := server.buckets.GetBucket(ctx, bucket, project.UUID)
	if err != nil {
		if storx.ErrBucketNotFound.Has(err) {
			sendJSONError(w, "bucket does not exist", "", http.StatusBadRequest)
		} else {
			sendJSONError(w, "unable to create geofence for bucket", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	b.Placement = placement

	_, err = server.buckets.UpdateBucket(ctx, b)
	if err != nil {
		switch {
		case storx.ErrBucketNotFound.Has(err):
			sendJSONError(w, "bucket does not exist", "", http.StatusBadRequest)
		case buckets.ErrBucketNotEmpty.Has(err):
			sendJSONError(w, "bucket must be empty", "", http.StatusBadRequest)
		default:
			sendJSONError(w, "unable to create geofence for bucket", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (server *Server) createGeofenceForBucket(w http.ResponseWriter, r *http.Request) {
	placement, err := parsePlacementConstraint(r.URL.Query().Get("region"))
	if err != nil {
		sendJSONError(w, err.Error(), "available: EU, EEA, US, DE", http.StatusBadRequest)
		return
	}

	server.updateBucket(w, r, placement)
}

func (server *Server) deleteGeofenceForBucket(w http.ResponseWriter, r *http.Request) {
	server.updateBucket(w, r, storx.EveryCountry)
}

func (server *Server) getBucketInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project, bucket, err := validateBucketPathParameters(mux.Vars(r))
	if err != nil {
		sendJSONError(w, err.Error(), "", http.StatusBadRequest)
		return
	}

	b, err := server.buckets.GetBucket(ctx, bucket, project.UUID)
	if err != nil {
		if storx.ErrBucketNotFound.Has(err) {
			sendJSONError(w, "bucket does not exist", "", http.StatusBadRequest)
		} else {
			sendJSONError(w, "unable to check bucket", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	data, err := json.Marshal(b)
	if err != nil {
		sendJSONError(w, "failed to marshal bucket", err.Error(), http.StatusInternalServerError)
	} else {
		sendJSONData(w, http.StatusOK, data)
	}
}
