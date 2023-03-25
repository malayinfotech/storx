// Copyright (C) 2018 Storx Labs, Inc.
// See LICENSE for copying information.

package satellitedb

import (
	"context"
	"errors"

	"common/uuid"
	"storx/satellite/console"
	"storx/satellite/satellitedb/dbx"
)

// ensures that resetPasswordTokens implements console.ResetPasswordTokens.
var _ console.ResetPasswordTokens = (*resetPasswordTokens)(nil)

type resetPasswordTokens struct {
	db dbx.Methods
}

// Create creates new reset password token.
func (rpt *resetPasswordTokens) Create(ctx context.Context, ownerID uuid.UUID) (_ *console.ResetPasswordToken, err error) {
	defer mon.Task()(&ctx)(&err)
	secret, err := console.NewResetPasswordSecret()
	if err != nil {
		return nil, err
	}

	resToken, err := rpt.db.Create_ResetPasswordToken(
		ctx,
		dbx.ResetPasswordToken_Secret(secret[:]),
		dbx.ResetPasswordToken_OwnerId(ownerID[:]),
	)
	if err != nil {
		return nil, err
	}

	return resetPasswordTokenFromDBX(ctx, resToken)
}

// GetBySecret retrieves ResetPasswordToken with given Secret.
func (rpt *resetPasswordTokens) GetBySecret(ctx context.Context, secret console.ResetPasswordSecret) (_ *console.ResetPasswordToken, err error) {
	defer mon.Task()(&ctx)(&err)
	resToken, err := rpt.db.Get_ResetPasswordToken_By_Secret(
		ctx,
		dbx.ResetPasswordToken_Secret(secret[:]),
	)
	if err != nil {
		return nil, err
	}

	return resetPasswordTokenFromDBX(ctx, resToken)
}

// GetByOwnerID retrieves ResetPasswordToken by ownerID.
func (rpt *resetPasswordTokens) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) (_ *console.ResetPasswordToken, err error) {
	defer mon.Task()(&ctx)(&err)
	resToken, err := rpt.db.Get_ResetPasswordToken_By_OwnerId(
		ctx,
		dbx.ResetPasswordToken_OwnerId(ownerID[:]),
	)
	if err != nil {
		return nil, err
	}

	return resetPasswordTokenFromDBX(ctx, resToken)
}

// Delete deletes ResetPasswordToken by ResetPasswordSecret.
func (rpt *resetPasswordTokens) Delete(ctx context.Context, secret console.ResetPasswordSecret) (err error) {
	defer mon.Task()(&ctx)(&err)
	_, err = rpt.db.Delete_ResetPasswordToken_By_Secret(
		ctx,
		dbx.ResetPasswordToken_Secret(secret[:]),
	)

	return err
}

// resetPasswordTokenFromDBX is used for creating ResetPasswordToken entity from autogenerated dbx.ResetPasswordToken struct.
func resetPasswordTokenFromDBX(ctx context.Context, resetToken *dbx.ResetPasswordToken) (_ *console.ResetPasswordToken, err error) {
	defer mon.Task()(&ctx)(&err)
	if resetToken == nil {
		return nil, errors.New("token parameter is nil")
	}

	var secret [32]byte

	copy(secret[:], resetToken.Secret)

	result := &console.ResetPasswordToken{
		Secret:    secret,
		OwnerID:   nil,
		CreatedAt: resetToken.CreatedAt,
	}

	if resetToken.OwnerId != nil {
		ownerID, err := uuid.FromBytes(resetToken.OwnerId)
		if err != nil {
			return nil, err
		}

		result.OwnerID = &ownerID
	}

	return result, nil
}