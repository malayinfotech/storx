#!/usr/bin/env bash

# Usage: VERSION=0.3.8 scripts/deploy-watchtower.sh
set -euo pipefail

: "${VERSION:?Must be set to the release version}"

docker manifest create storxlabs/watchtower:latest \
storxlabs/watchtower:i386-${VERSION} \
storxlabs/watchtower:amd64-${VERSION} \
storxlabs/watchtower:arm64v8-${VERSION} \
storxlabs/watchtower:armhf-${VERSION}

docker manifest annotate storxlabs/watchtower:latest \
storxlabs/watchtower:i386-${VERSION} --os linux --arch 386

docker manifest annotate storxlabs/watchtower:latest \
storxlabs/watchtower:amd64-${VERSION} --os linux --arch amd64

docker manifest annotate storxlabs/watchtower:latest \
storxlabs/watchtower:arm64v8-${VERSION} --os linux --arch arm64 --variant v8

docker manifest annotate storxlabs/watchtower:latest \
storxlabs/watchtower:armhf-${VERSION} --os linux --arch arm

docker manifest push --purge storxlabs/watchtower:latest
