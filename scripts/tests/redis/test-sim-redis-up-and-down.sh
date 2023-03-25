#!/usr/bin/env bash
set -Eeo pipefail
set +x

# Required environment variables
if [ -z "${STORX_SIM_POSTGRES}" ]; then
	echo "STORX_SIM_POSTGRES environment variable must be set to a non-empty string"
	exit 1
fi

if [ -z "${STORX_REDIS_PORT}" ]; then
	echo STORX_REDIS_PORT env var is required
	exit 1
fi

# constants
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
readonly SCRIPT_DIR
TMP_DIR=$(mktemp -d -t tmp.XXXXXXXXXX)
readonly TMP_DIR
STORX_REDIS_DIR=$(mktemp -d -p /tmp test-sim-redis.XXXX)
readonly STORX_REDIS_DIR
export STORX_REDIS_DIR

cleanup() {
	trap - EXIT ERR

	"${SCRIPT_DIR}/redis-server.sh" stop
	rm -rf "${TMP_DIR}"
	rm -rf "${STORX_REDIS_DIR}"
}
trap cleanup ERR EXIT

echo "install sim"
make -C "$SCRIPT_DIR"/../../.. install-sim

echo "overriding default max segment size to 6MiB"
GOBIN="${TMP_DIR}" go install -v -ldflags "-X 'storx/uplink.maxSegmentSize=6MiB'" storx/storx/cmd/uplink

# use modified version of uplink
export PATH="${TMP_DIR}:${PATH}"
export STORX_NETWORK_DIR="${TMP_DIR}"

STORX_NETWORK_HOST4=${STORX_NETWORK_HOST4:-127.0.0.1}
export STORX_REDIS_HOST=${STORX_NETWORK_HOST4}

# setup the network
"${SCRIPT_DIR}/redis-server.sh" start
storx-sim --failfast -x --satellites 1 --host "${STORX_NETWORK_HOST4}" network \
	--postgres="${STORX_SIM_POSTGRES}" --redis="${STORX_REDIS_HOST}:${STORX_REDIS_PORT}" setup

# run test that checks that the satellite runs when Redis is up and down
storx-sim --failfast -x --satellites 1 --host "${STORX_NETWORK_HOST4}" network \
	--redis="127.0.0.1:6379" test bash "${SCRIPT_DIR}/test-uplink-redis-up-and-down.sh" "${REDIS_CONTAINER_NAME}"

# run test that checks that the satellite runs despite of not being able to connect to Redis
"${SCRIPT_DIR}/redis-server.sh" stop
storx-sim --failfast -x --satellites 1 --host "${STORX_NETWORK_HOST4}" network \
	--redis="127.0.0.1:6379" test bash "${SCRIPT_DIR}/../integration/test-uplink.sh"
