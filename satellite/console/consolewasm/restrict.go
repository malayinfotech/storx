// Copyright (C) 2020 Storx Labs, Inc.
// See LICENSE for copying information.

package consolewasm

import (
	"strings"

	"common/grant"
)

// RestrictGrant restricts an access grant with the permissions and paths and returns a new access grant.
func RestrictGrant(accessGrant string, paths []string, permission Permission) (string, error) {
	access, err := grant.ParseAccess(accessGrant)
	if err != nil {
		return "", err
	}

	prefixes := make([]grant.SharePrefix, 0, len(paths))
	for _, path := range paths {
		parts := strings.SplitN(path, "/", 2)
		prefix := grant.SharePrefix{Bucket: parts[0]}
		if len(parts) > 1 {
			prefix.Prefix = parts[1]
		}
		prefixes = append(prefixes, prefix)
	}

	restricted, err := access.Restrict(grant.Permission(permission), prefixes...)
	if err != nil {
		return "", err
	}

	return restricted.Serialize()
}
