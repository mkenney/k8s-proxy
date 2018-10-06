#!/bin/sh
set -e

WORKDIR=$(pwd)
exit_code=0

go get -v github.com/golang/lint/golint
[ "0" = "$?" ] || exit 1

env GO111MODULE=on go build  ./...

for dir in $(go list ./... | grep -v vendor); do
    echo "golint $dir"
    result=$(golint $dir)
    if [ "" != "$result" ]; then
        echo $result
        exit_code=5
    fi
    if [ "0" != "$exit_code" ]; then
        exit $exit_code
    fi
done

rm -f coverage.txt
for dir in $(go list ./... | grep -v vendor); do
    go test -timeout 20s -coverprofile=profile.out $dir
    exit_code=$?
    if [ "0" != "$exit_code" ]; then
        exit $exit_code
    fi
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

exit $exit_code
