#!/usr/bin/env bash
# Copyright (C) 2022 Storx Labs, Inc.
# See LICENSE for copying information.
set -euxo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

npm install --prefer-offline --no-audit --logleve verbose
echo "module stub" > ./node_modules/go.mod # prevent Go from scanning this dir
npm run build

npm run lint-ci
npm audit || true
npm run test
