#!/usr/bin/env bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

source $SCRIPTDIR/../postgres-dev.sh

export STORX_MIGRATION_DB="${STORX_SIM_POSTGRES}&options=--search_path=satellite/0/meta"

$SCRIPTDIR/test-sim-backwards.sh