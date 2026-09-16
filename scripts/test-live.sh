#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
: "${HEADROOM_TEST_URL:?Set HEADROOM_TEST_URL to the compression endpoint}"
docker run --rm --user "$(id -u):$(id -g)" -e GOCACHE=/tmp/go-cache -e GOPATH=/tmp/go --network "${HEADROOM_NETWORK:-llm}" -e HEADROOM_TEST_URL -v "$PWD:/src" -w /src golang:1.26-bookworm go test -v -run TestLiveHeadroom .
