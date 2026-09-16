#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export REPOSITORY_URL="${REPOSITORY_URL:-https://github.com/${GITHUB_REPOSITORY:-frankyw/cliproxyapi-headroom}}"
export PLUGIN_AUTHOR="${PLUGIN_AUTHOR:-${GITHUB_REPOSITORY_OWNER:-frankyw}}"
docker run --rm --user "$(id -u):$(id -g)" -e GOCACHE=/tmp/go-cache -e GOPATH=/tmp/go -e REPOSITORY_URL -e PLUGIN_AUTHOR -v "$PWD:/src" -w /src golang:1.26-bookworm sh -ec 'go mod download; test -z "$(gofmt -l *.go)"; go test -race ./...; mkdir -p dist; CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w -X main.repository=$REPOSITORY_URL -X main.author=$PLUGIN_AUTHOR" -o dist/headroom.so .'
python3 scripts/package.py
