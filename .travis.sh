#!/bin/sh
set -e

docker run \
    --rm \
    -v $(pwd):/go/src/github.com/mkenney/k8s-proxy \
    --entrypoint="/go/src/github.com/mkenney/k8s-proxy/.travis.entrypoint.sh" \
    golang:1.10-alpine
