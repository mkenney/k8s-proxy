#!/bin/sh
set -e
echo "" > coverage.txt

# the alpine image doesn't come with git
apk update && apk add git

go get -v github.com/golang/lint/golint
[ "0" = "$?" ] || exit 1

cd /go/src/github.com/mkenney/k8s-proxy/pkg
[ "0" = "$?" ] || exit 2

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

exit_code=0
for dir in $(go list ./... | grep -v vendor); do
    echo "golint $dir"
    result=$(golint $dir)
    if [ "" != "$result" ]; then
        echo $result
        exit_code=4
    fi
done

exit $exit_code
