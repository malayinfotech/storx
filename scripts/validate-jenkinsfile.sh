#!/usr/bin/env bash
set -euxo pipefail
curl -X POST -F "jenkinsfile=<$1" https://build.dev.storx/pipeline-model-converter/validate
