#!/usr/bin/env bash

# Usage: TAG=6e8c4ed-v0.19.0-go1.12.9 scripts/deploy-storagenode.sh
set -euo pipefail

: "${TAG:?Must be set to the gitish version of the release without architecture}"

for v in alpha arm beta latest; do
  for app in multinode storagenode; do
	  docker manifest create --amend storxlabs/$app:$v \
    storxlabs/$app:${TAG}-amd64 \
    storxlabs/$app:${TAG}-arm32v5 \
    storxlabs/$app:${TAG}-arm64v8

    docker manifest annotate storxlabs/$app:$v \
    storxlabs/$app:${TAG}-amd64 --os linux --arch amd64

    docker manifest annotate storxlabs/$app:$v \
    storxlabs/$app:${TAG}-arm32v5 --os linux --arch arm --variant v5

    docker manifest annotate storxlabs/$app:$v \
    storxlabs/$app:${TAG}-arm64v8 --os linux --arch arm64 --variant v8

    docker manifest push --purge storxlabs/$app:$v
  done
done
