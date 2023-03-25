// Copyright (C) 2019 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/zeebo/errs"

	"common/macaroon"
	"common/storx"
	"common/uuid"
	"storx/satellite/buckets"
	"storx/satellite/metabase"
	"storx/satellite/satellitedb/dbx"
)

type bucketsDB struct {
	db *satelliteDB
}

// CreateBucket creates a new bucket.
func (db *bucketsDB) CreateBucket(ctx context.Context, bucket storx.Bucket) (_ storx.Bucket, err error) {
	defer mon.Task()(&ctx)(&err)

	optionalFields := dbx.BucketMetainfo_Create_Fields{}
	if !bucket.PartnerID.IsZero() || bucket.UserAgent != nil {
		optionalFields = dbx.BucketMetainfo_Create_Fields{
			PartnerId: dbx.BucketMetainfo_PartnerId(bucket.PartnerID[:]),
			UserAgent: dbx.BucketMetainfo_UserAgent(bucket.UserAgent),
		}
	}
	optionalFields.Placement = dbx.BucketMetainfo_Placement(int(bucket.Placement))

	row, err := db.db.Create_BucketMetainfo(ctx,
		dbx.BucketMetainfo_Id(bucket.ID[:]),
		dbx.BucketMetainfo_ProjectId(bucket.ProjectID[:]),
		dbx.BucketMetainfo_Name([]byte(bucket.Name)),
		dbx.BucketMetainfo_PathCipher(int(bucket.PathCipher)),
		dbx.BucketMetainfo_DefaultSegmentSize(int(bucket.DefaultSegmentsSize)),
		dbx.BucketMetainfo_DefaultEncryptionCipherSuite(int(bucket.DefaultEncryptionParameters.CipherSuite)),
		dbx.BucketMetainfo_DefaultEncryptionBlockSize(int(bucket.DefaultEncryptionParameters.BlockSize)),
		dbx.BucketMetainfo_DefaultRedundancyAlgorithm(int(bucket.DefaultRedundancyScheme.Algorithm)),
		dbx.BucketMetainfo_DefaultRedundancyShareSize(int(bucket.DefaultRedundancyScheme.ShareSize)),
		dbx.BucketMetainfo_DefaultRedundancyRequiredShares(int(bucket.DefaultRedundancyScheme.RequiredShares)),
		dbx.BucketMetainfo_DefaultRedundancyRepairShares(int(bucket.DefaultRedundancyScheme.RepairShares)),
		dbx.BucketMetainfo_DefaultRedundancyOptimalShares(int(bucket.DefaultRedundancyScheme.OptimalShares)),
		dbx.BucketMetainfo_DefaultRedundancyTotalShares(int(bucket.DefaultRedundancyScheme.TotalShares)),
		optionalFields,
	)
	if err != nil {
		return storx.Bucket{}, storx.ErrBucket.Wrap(err)
	}

	bucket, err = convertDBXtoBucket(row)
	if err != nil {
		return storx.Bucket{}, storx.ErrBucket.Wrap(err)
	}
	return bucket, nil
}

// GetBucket returns a bucket.
func (db *bucketsDB) GetBucket(ctx context.Context, bucketName []byte, projectID uuid.UUID) (_ storx.Bucket, err error) {
	defer mon.Task()(&ctx)(&err)
	dbxBucket, err := db.db.Get_BucketMetainfo_By_ProjectId_And_Name(ctx,
		dbx.BucketMetainfo_ProjectId(projectID[:]),
		dbx.BucketMetainfo_Name(bucketName),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storx.Bucket{}, storx.ErrBucketNotFound.New("%s", bucketName)
		}
		return storx.Bucket{}, storx.ErrBucket.Wrap(err)
	}
	return convertDBXtoBucket(dbxBucket)
}

// GetBucketPlacement returns with the placement constraint identifier.
func (db *bucketsDB) GetBucketPlacement(ctx context.Context, bucketName []byte, projectID uuid.UUID) (placement storx.PlacementConstraint, err error) {
	defer mon.Task()(&ctx)(&err)
	dbxPlacement, err := db.db.Get_BucketMetainfo_Placement_By_ProjectId_And_Name(ctx,
		dbx.BucketMetainfo_ProjectId(projectID[:]),
		dbx.BucketMetainfo_Name(bucketName),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storx.EveryCountry, storx.ErrBucketNotFound.New("%s", bucketName)
		}
		return storx.EveryCountry, storx.ErrBucket.Wrap(err)
	}
	placement = storx.EveryCountry
	if dbxPlacement.Placement != nil {
		placement = storx.PlacementConstraint(*dbxPlacement.Placement)
	}

	return placement, nil
}

// GetMinimalBucket returns existing bucket with minimal number of fields.
func (db *bucketsDB) GetMinimalBucket(ctx context.Context, bucketName []byte, projectID uuid.UUID) (_ buckets.Bucket, err error) {
	defer mon.Task()(&ctx)(&err)
	row, err := db.db.Get_BucketMetainfo_CreatedAt_By_ProjectId_And_Name(ctx,
		dbx.BucketMetainfo_ProjectId(projectID[:]),
		dbx.BucketMetainfo_Name(bucketName),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return buckets.Bucket{}, storx.ErrBucketNotFound.New("%s", bucketName)
		}
		return buckets.Bucket{}, storx.ErrBucket.Wrap(err)
	}
	return buckets.Bucket{
		Name:      bucketName,
		CreatedAt: row.CreatedAt,
	}, nil
}

// HasBucket returns if a bucket exists.
func (db *bucketsDB) HasBucket(ctx context.Context, bucketName []byte, projectID uuid.UUID) (exists bool, err error) {
	defer mon.Task()(&ctx)(&err)

	exists, err = db.db.Has_BucketMetainfo_By_ProjectId_And_Name(ctx,
		dbx.BucketMetainfo_ProjectId(projectID[:]),
		dbx.BucketMetainfo_Name(bucketName),
	)
	return exists, storx.ErrBucket.Wrap(err)
}

// GetBucketID returns an existing bucket id.
func (db *bucketsDB) GetBucketID(ctx context.Context, bucket metabase.BucketLocation) (_ uuid.UUID, err error) {
	defer mon.Task()(&ctx)(&err)
	dbxID, err := db.db.Get_BucketMetainfo_Id_By_ProjectId_And_Name(ctx,
		dbx.BucketMetainfo_ProjectId(bucket.ProjectID[:]),
		dbx.BucketMetainfo_Name([]byte(bucket.BucketName)),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.UUID{}, storx.ErrBucketNotFound.New("%s", bucket.BucketName)
		}
		return uuid.UUID{}, storx.ErrBucket.Wrap(err)
	}

	id, err := uuid.FromBytes(dbxID.Id)
	if err != nil {
		return id, storx.ErrBucket.Wrap(err)
	}
	return id, err
}

// UpdateBucket updates a bucket.
func (db *bucketsDB) UpdateBucket(ctx context.Context, bucket storx.Bucket) (_ storx.Bucket, err error) {
	defer mon.Task()(&ctx)(&err)

	var updateFields dbx.BucketMetainfo_Update_Fields
	if !bucket.PartnerID.IsZero() {
		updateFields.PartnerId = dbx.BucketMetainfo_PartnerId(bucket.PartnerID[:])
	}

	if bucket.UserAgent != nil {
		updateFields.UserAgent = dbx.BucketMetainfo_UserAgent(bucket.UserAgent)
	}

	updateFields.Placement = dbx.BucketMetainfo_Placement(int(bucket.Placement))

	dbxBucket, err := db.db.Update_BucketMetainfo_By_ProjectId_And_Name(ctx, dbx.BucketMetainfo_ProjectId(bucket.ProjectID[:]), dbx.BucketMetainfo_Name([]byte(bucket.Name)), updateFields)
	if err != nil {
		return storx.Bucket{}, storx.ErrBucket.Wrap(err)
	}
	return convertDBXtoBucket(dbxBucket)
}

// DeleteBucket deletes a bucket.
func (db *bucketsDB) DeleteBucket(ctx context.Context, bucketName []byte, projectID uuid.UUID) (err error) {
	defer mon.Task()(&ctx)(&err)
	deleted, err := db.db.Delete_BucketMetainfo_By_ProjectId_And_Name(ctx,
		dbx.BucketMetainfo_ProjectId(projectID[:]),
		dbx.BucketMetainfo_Name(bucketName),
	)
	if err != nil {
		return storx.ErrBucket.Wrap(err)
	}
	if !deleted {
		return storx.ErrBucketNotFound.New("%s", bucketName)
	}
	return nil
}

// ListBuckets returns a list of buckets for a project.
func (db *bucketsDB) ListBuckets(ctx context.Context, projectID uuid.UUID, listOpts storx.BucketListOptions, allowedBuckets macaroon.AllowedBuckets) (bucketList storx.BucketList, err error) {
	defer mon.Task()(&ctx)(&err)

	const defaultListLimit = 10000
	if listOpts.Limit < 1 {
		listOpts.Limit = defaultListLimit
	}
	limit := listOpts.Limit + 1 // add one to detect More

	for {
		var dbxBuckets []*dbx.BucketMetainfo
		switch listOpts.Direction {
		// For simplictiy we are only supporting the forward direction for listing buckets
		case storx.Forward:
			dbxBuckets, err = db.db.Limited_BucketMetainfo_By_ProjectId_And_Name_GreaterOrEqual_OrderBy_Asc_Name(ctx,
				dbx.BucketMetainfo_ProjectId(projectID[:]),
				dbx.BucketMetainfo_Name([]byte(listOpts.Cursor)),
				limit,
				0,
			)

		// After is only called by BucketListOptions.NextPage and is the paginated Forward direction
		case storx.After:
			dbxBuckets, err = db.db.Limited_BucketMetainfo_By_ProjectId_And_Name_Greater_OrderBy_Asc_Name(ctx,
				dbx.BucketMetainfo_ProjectId(projectID[:]),
				dbx.BucketMetainfo_Name([]byte(listOpts.Cursor)),
				limit,
				0,
			)
		default:
			return bucketList, errors.New("unknown list direction")
		}
		if err != nil {
			return bucketList, storx.ErrBucket.Wrap(err)
		}

		bucketList.More = len(dbxBuckets) > listOpts.Limit
		if bucketList.More {
			// If there are more buckets than listOpts.limit returned,
			// then remove the extra buckets so that we do not return
			// more then the limit
			dbxBuckets = dbxBuckets[0:listOpts.Limit]
		}

		if bucketList.Items == nil {
			bucketList.Items = make([]storx.Bucket, 0, len(dbxBuckets))
		}

		for _, dbxBucket := range dbxBuckets {
			// Check that the bucket is allowed to be viewed
			_, bucketAllowed := allowedBuckets.Buckets[string(dbxBucket.Name)]
			if bucketAllowed || allowedBuckets.All {
				item, err := convertDBXtoBucket(dbxBucket)
				if err != nil {
					return bucketList, storx.ErrBucket.Wrap(err)
				}
				bucketList.Items = append(bucketList.Items, item)
			}
		}

		if len(bucketList.Items) < listOpts.Limit && bucketList.More {
			// If we filtered out disallowed buckets, then get more buckets
			// out of database so that we return `limit` number of buckets
			listOpts = storx.BucketListOptions{
				Cursor:    string(dbxBuckets[len(dbxBuckets)-1].Name),
				Limit:     listOpts.Limit,
				Direction: storx.After,
			}
			continue
		}
		break
	}

	return bucketList, nil
}

// CountBuckets returns the number of buckets a project currently has.
func (db *bucketsDB) CountBuckets(ctx context.Context, projectID uuid.UUID) (count int, err error) {
	count64, err := db.db.Count_BucketMetainfo_Name_By_ProjectId(ctx, dbx.BucketMetainfo_ProjectId(projectID[:]))
	if err != nil {
		return -1, err
	}
	return int(count64), nil
}

func convertDBXtoBucket(dbxBucket *dbx.BucketMetainfo) (bucket storx.Bucket, err error) {
	id, err := uuid.FromBytes(dbxBucket.Id)
	if err != nil {
		return bucket, storx.ErrBucket.Wrap(err)
	}
	project, err := uuid.FromBytes(dbxBucket.ProjectId)
	if err != nil {
		return bucket, storx.ErrBucket.Wrap(err)
	}

	bucket = storx.Bucket{
		ID:                  id,
		Name:                string(dbxBucket.Name),
		ProjectID:           project,
		Created:             dbxBucket.CreatedAt,
		PathCipher:          storx.CipherSuite(dbxBucket.PathCipher),
		DefaultSegmentsSize: int64(dbxBucket.DefaultSegmentSize),
		DefaultRedundancyScheme: storx.RedundancyScheme{
			Algorithm:      storx.RedundancyAlgorithm(dbxBucket.DefaultRedundancyAlgorithm),
			ShareSize:      int32(dbxBucket.DefaultRedundancyShareSize),
			RequiredShares: int16(dbxBucket.DefaultRedundancyRequiredShares),
			RepairShares:   int16(dbxBucket.DefaultRedundancyRepairShares),
			OptimalShares:  int16(dbxBucket.DefaultRedundancyOptimalShares),
			TotalShares:    int16(dbxBucket.DefaultRedundancyTotalShares),
		},
		DefaultEncryptionParameters: storx.EncryptionParameters{
			CipherSuite: storx.CipherSuite(dbxBucket.DefaultEncryptionCipherSuite),
			BlockSize:   int32(dbxBucket.DefaultEncryptionBlockSize),
		},
	}

	if dbxBucket.Placement != nil {
		bucket.Placement = storx.PlacementConstraint(*dbxBucket.Placement)
	}

	if dbxBucket.PartnerId != nil {
		partnerID, err := uuid.FromBytes(dbxBucket.PartnerId)
		if err != nil {
			return bucket, storx.ErrBucket.Wrap(err)
		}
		bucket.PartnerID = partnerID
	}

	if dbxBucket.UserAgent != nil {
		bucket.UserAgent = dbxBucket.UserAgent
	}

	return bucket, nil
}

// IterateBucketLocations iterates through all buckets from some point with limit.
func (db *bucketsDB) IterateBucketLocations(ctx context.Context, projectID uuid.UUID, bucketName string, limit int, fn func([]metabase.BucketLocation) error) (more bool, err error) {
	defer mon.Task()(&ctx)(&err)

	var result []metabase.BucketLocation

	moreLimit := limit + 1
	rows, err := db.db.QueryContext(ctx, `
			SELECT project_id, name
			FROM bucket_metainfos
			WHERE (project_id, name) > ($1, $2)
			GROUP BY (project_id, name)
			ORDER BY (project_id, name) ASC LIMIT $3
	`, projectID, bucketName, moreLimit)
	if err != nil {
		return false, storx.ErrBucket.New("BatchBuckets query error: %s", err)
	}
	defer func() {
		err = errs.Combine(err, Error.Wrap(rows.Close()))
	}()

	for rows.Next() {
		var bucketLocation metabase.BucketLocation

		if err = rows.Scan(&bucketLocation.ProjectID, &bucketLocation.BucketName); err != nil {
			return false, storx.ErrBucket.New("bucket location scan error: %s", err)
		}

		result = append(result, bucketLocation)
	}

	if err = rows.Err(); err != nil {
		return false, storx.ErrBucket.Wrap(err)
	}

	if len(result) == 0 {
		return false, nil
	}

	if len(result) > limit {
		return true, Error.Wrap(fn(result[:len(result)-1]))
	}

	return false, Error.Wrap(fn(result))
}
