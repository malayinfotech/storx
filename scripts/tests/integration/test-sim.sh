#!/usr/bin/env bash
set -ueo pipefail
set +x

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# setup tmpdir for testfiles and cleanup
TMP=$(mktemp -d -t tmp.XXXXXXXXXX)
cleanup(){
	rm -rf "$TMP"
}
trap cleanup EXIT

echo "Running test-sim"
make -C "$SCRIPTDIR"/../../.. install-sim

echo "Overriding default max segment size to 6MiB"
GOBIN=$TMP go install -v -ldflags "-X 'storx/uplink.maxSegmentSize=6MiB'" storx/storx/cmd/uplink

# use modified version of uplink
export PATH=$TMP:$PATH

export STORX_NETWORK_DIR=$TMP

STORX_NETWORK_HOST4=${STORX_NETWORK_HOST4:-127.0.0.1}
STORX_SIM_POSTGRES=${STORX_SIM_POSTGRES:-""}

# setup the network
# if postgres connection string is set as STORX_SIM_POSTGRES then use that for testing
if [ -z ${STORX_SIM_POSTGRES} ]; then
	storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network setup
else
	storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network --postgres=$STORX_SIM_POSTGRES setup
fi

# run tests
storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network test bash "$SCRIPTDIR"/test-uplink.sh
storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network test bash "$SCRIPTDIR"/test-uplink-share.sh
storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network test bash "$SCRIPTDIR"/test-billing.sh

storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network test bash "$SCRIPTDIR"/test-uplink-rs-upload.sh
# change RS values and try download 
sed -i 's@# metainfo.rs: 4/6/8/10-256 B@metainfo.rs: 2/3/6/8-256 B@g' $(storx-sim network env SATELLITE_0_DIR)/config.yaml
storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network test bash "$SCRIPTDIR"/test-uplink-rs-download.sh

storx-sim -x --satellites 1 --host $STORX_NETWORK_HOST4 network destroy

# setup the network with ipv6
#storx-sim -x --host "::1" network setup
# aws-cli doesn't support gateway with ipv6 address, so change it to use localhost
#find "$STORX_NETWORK_DIR"/gateway -type f -name config.yaml -exec sed -i 's/server.address: "\[::1\]/server.address: "127.0.0.1/' '{}' +
# run aws-cli tests using ipv6
#storx-sim -x --host "::1" network test bash "$SCRIPTDIR"/test-sim-aws.sh
#storx-sim -x network destroy
