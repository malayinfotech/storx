#!/usr/bin/env bash

SCRIPTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
export STORX_REDIS_PORT=7379

# shellcheck source=/postgres-dev.sh
source "${SCRIPTDIR}/../postgres-dev.sh"

"${SCRIPTDIR}/test-sim-redis-up-and-down.sh"