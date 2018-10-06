#!/bin/sh
set -e

GO111MODULE=on
GOFLAGS=-mod-vendor

WORKDIR=$(pwd)
exit_code=0

echo "go build ./..."
go build  ./...
if [ "0" != "$?" ]; then
    exit 10
fi

for dir in $(go list ./... | grep -v vendor); do
    echo "golint $dir"
    result=$(GO111MODULE=on golint $dir)
    if [ "" != "$result" ]; then
        echo $result
        exit 20
    fi
done

rm -f coverage.txt
for dir in $(go list ./... | grep -v vendor); do
    go test -mod=vendor -timeout=20s -coverprofile=profile.out $dir
    exit_code=$?
    if [ "0" != "$exit_code" ]; then
        exit 30
    fi
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

exit $exit_code
