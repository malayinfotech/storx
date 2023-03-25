// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package consolewasm

import (
	"encoding/base64"

	"common/encryption"
	"common/grant"
	"common/macaroon"
	"common/storx"
)

// GenAccessGrant creates a new access grant and returns it serialized form.
func GenAccessGrant(satelliteNodeURL, apiKey, encryptionPassphrase, base64EncodedSalt string) (string, error) {
	parsedAPIKey, err := macaroon.ParseAPIKey(apiKey)
	if err != nil {
		return "", err
	}

	key, err := DeriveRootKey(encryptionPassphrase, base64EncodedSalt)
	if err != nil {
		return "", err
	}

	encAccess := grant.NewEncryptionAccessWithDefaultKey(key)
	encAccess.SetDefaultPathCipher(storx.EncAESGCM)
	encAccess.LimitTo(parsedAPIKey)

	accessString, err := (&grant.Access{
		SatelliteAddress: satelliteNodeURL,
		APIKey:           parsedAPIKey,
		EncAccess:        encAccess,
	}).Serialize()
	if err != nil {
		return "", err
	}
	return accessString, nil
}

// DeriveRootKey derives the root key portion of the access grant.
func DeriveRootKey(encryptionPassphrase, base64EncodedSalt string) (*storx.Key, error) {
	const concurrency = 8
	saltBytes, err := base64.StdEncoding.DecodeString(base64EncodedSalt)
	if err != nil {
		return nil, err
	}
	return encryption.DeriveRootKey([]byte(encryptionPassphrase), saltBytes, "", concurrency)
}
