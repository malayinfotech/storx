VERSION 0.6
FROM golang:1.18
WORKDIR /go/storx

multinode-web:
    FROM node:18
    WORKDIR /build
    COPY web/multinode .
    RUN ./build.sh
    SAVE ARTIFACT dist AS LOCAL web/storagenode/dist

wasm:
   FROM storxlabs/ci
   ENV TAG=earthly
   COPY go.mod .
   COPY go.sum .
   COPY satellite/console/wasm satellite/console/wasm
   COPY satellite/console/consolewasm satellite/console/consolewasm
   COPY scripts scripts
   RUN scripts/build-wasm.sh
   SAVE ARTIFACT release/earthly/wasm wasm AS LOCAL web/satellite/static/wasm

storagenode-web:
    FROM node:18
    WORKDIR /build
    COPY web/storagenode .
    RUN ./build.sh
    SAVE ARTIFACT dist AS LOCAL web/storagenode/dist
    SAVE ARTIFACT static AS LOCAL web/storagenode/static

satellite-web:
    FROM node:18
    WORKDIR /build
    COPY web/satellite .
    RUN ./build.sh
    COPY +wasm/wasm static/wasm
    SAVE ARTIFACT dist AS LOCAL web/satellite/dist
    SAVE ARTIFACT static AS LOCAL web/satellite/static

storagenode-bin:
    COPY go.mod go.mod
    COPY go.sum go.sum
    COPY private private
    COPY cmd/storagenode cmd/storagenode
    COPY storage storage
    COPY storagenode storagenode
    COPY multinode multinode
    COPY web/storagenode web/storagenode
    RUN --mount=type=cache,target=/root/.cache/go-build \
        --mount=type=cache,target=/go/pkg/mod \
        go build -o release/earthly/storagenode storx/storx/cmd/storagenode
    SAVE ARTIFACT release/earthly binaries AS LOCAL release/earthly

build-binaries:
    COPY . .
    RUN --mount=type=cache,target=/root/.cache/go-build \
        --mount=type=cache,target=/go/pkg/mod \
        go build -o release/earthly/satellite storx/storx/cmd/satellite
    RUN --mount=type=cache,target=/root/.cache/go-build \
        --mount=type=cache,target=/go/pkg/mod \
        go build -o release/earthly/uplink storx/storx/cmd/uplink
    RUN --mount=type=cache,target=/root/.cache/go-build \
        --mount=type=cache,target=/go/pkg/mod \
        go build -o release/earthly/identity storx/storx/cmd/identity
    SAVE ARTIFACT release/earthly binaries AS LOCAL release/earthly

build-storxup:
    RUN --mount=type=cache,target=/root/.cache/go-build \
        --mount=type=cache,target=/go/pkg/mod \
        CGO_ENABLED=0 go install storx/storx-up@main
    SAVE ARTIFACT /go/bin binaries AS LOCAL dist/up


deploy-remote:
    FROM ubuntu
    RUN apt-get update && apt-get install -y git wget unzip
    RUN cd /tmp && wget https://releases.hashicorp.com/nomad/1.3.5/nomad_1.3.5_linux_amd64.zip -O nomad.zip && unzip nomad.zip && mv nomad /usr/local/bin && rm nomad.zip
    COPY +build-storxup/binaries /usr/local/bin
    COPY .git .git
    ARG TAG=$(git rev-parse --short HEAD)
    ARG IMAGE=img.dev.storx/dev/storx
    BUILD +build-tagged-image --TAG=$TAG --IMAGE=$IMAGE
    ARG --required nomad
    ARG --required ip
    COPY scripts/deploy/deploy-nightly.sh .
    ENV NOMAD_ADDR=$nomad
    ENV IP=$ip
    ENV IMAGE=$IMAGE
    ENV TAG=$TAG
    RUN --push ./deploy-nightly.sh

deploy-local:
    COPY +build-storxup/binaries /usr/local/bin
    COPY .git .git
    ARG TAG=$(git rev-parse --short HEAD)
    ARG IMAGE=img.dev.storx/dev/storx
    BUILD +build-tagged-image --TAG=$TAG --IMAGE=$IMAGE
    WORKDIR /opt/storx-up
    RUN storx-up init db,minimal,edge
    RUN storx-up image satellite-api,storagenode $IMAGE:$TAG
    SAVE ARTIFACT /opt/storx-up/docker-compose.yaml compose AS LOCAL docker-compose.yaml

build-image:
    FROM storxlabs/ci
    COPY .git .git
    ARG IMAGE=img.dev.storx/dev/storx
    ARG TAG=$(git rev-parse --short HEAD)
    BUILD +build-tagged-image --TAG=$TAG --IMAGE=$IMAGE

build-tagged-image:
    ARG --required TAG
    ARG --required IMAGE
    FROM img.dev.storx/storxup/base:20230208-1
    COPY +multinode-web/dist /var/lib/storx/storx/web/multinode/dist
    COPY +satellite-web/dist /var/lib/storx/storx/web/satellite/dist
    COPY +satellite-web/static /var/lib/storx/storx/web/satellite/static
    COPY +storagenode-web/dist /var/lib/storx/storx/web/storagenode/dist
    COPY +storagenode-web/static /var/lib/storx/storx/web/storagenode/static
    COPY +build-binaries/binaries /var/lib/storx/go/bin/
    COPY +storagenode-bin/binaries /var/lib/storx/go/bin/
    COPY +build-storxup/binaries  /var/lib/storx/go/bin/
    SAVE IMAGE --push $IMAGE:$TAG

run:
    LOCALLY
    RUN docker-compose up

test:
   COPY . .
   RUN go install github.com/mfridman/tparse@36f80740879e24ba6695649290a240c5908ffcbb
   RUN mkdir build
   RUN --mount=type=cache,target=/root/.cache/go-build \
       --mount=type=cache,target=/go/pkg/mod \
       go test -json ./... | tee build/tests.json
   SAVE ARTIFACT build/tests.json AS LOCAL build/tests.json

integration:
   COPY +build/storx-up /usr/local/bin/storx-up
   COPY test/test.sh .
   WITH DOCKER
      RUN ./test.sh
   END

check-format:
   COPY . .
   RUN mkdir build
   RUN bash -c '[[ $(git status --short) == "" ]] || (echo "Before formatting, please commit all your work!!! (Formatter will format only last commit)" && exit -1)'
   RUN git show --name-only --pretty=format: | grep ".go" | xargs --no-run-if-empty -n1 gofmt -s -w
   RUN git diff > build/format.patch
   SAVE ARTIFACT build/format.patch

format:
   LOCALLY
   COPY +check-format/format.patch build/format.patch
   RUN git apply --allow-empty build/format.patch
   RUN git status
