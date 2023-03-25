#!/usr/bin/env bash
set -ex
storx-up init nomad --name=core --ip=$IP minimal,gc
storx-up image satellite-api,storagenode,gc $IMAGE:$TAG
storx-up persist storagenode,satellite-api,gc
storx-up env set satellite-api STORX_DATABASE_OPTIONS_MIGRATION_UNSAFE=full,testdata
nomad run storx.hcl
