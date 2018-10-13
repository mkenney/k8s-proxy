#!/bin/sh
set -e

export GO111MODULE=on
export GOFLAGS=-mod-vendor

WORKDIR=$(pwd)
exit_code=0

#echo "go build ./..."
#go build  ./...
#if [ "0" != "$?" ]; then
#    exit 10
#fi

#go get -v github.com/golang/lint/golint
#[ "0" = "$?" ] || exit 10
#
#for dir in $(go list ./... | grep -v vendor); do
#    echo "golint $dir"
#    result=$(GO111MODULE=on golint $dir)
#    if [ "" != "$result" ]; then
#        echo $result
#        exit 20
#    fi
#done

rm -f coverage.txt
for dir in $(go list ./... | grep -v vendor); do
    go test -timeout=20s -coverprofile=profile.out $dir
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
