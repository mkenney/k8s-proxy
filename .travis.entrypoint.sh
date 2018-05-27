#!/bin/sh
set -e
echo "" > coverage.txt

SRC=/go/src/github.com/mkenney/k8s-proxy

# the alpine image doesn't come with git
apk update && apk add git

go get -v github.com/golang/lint/golint
[ "0" = "$?" ] || exit 1

cd $SRC
[ "0" = "$?" ] || exit 2

rm -f $SRC/coverage.txt
for dir in $(go list ./...); do
    echo "go test -timeout 20s -coverprofile=$SRC/profile.out $dir"
    go test -timeout 20s -coverprofile=$SRC/profile.out $dir
    exit_code=$?
    if [ "0" != "$exit_code" ]; then
        exit $exit_code
    fi
    if [ -f $SRC/profile.out ]; then
        cat $SRC/profile.out >> $SRC/coverage.txt
        rm $SRC/profile.out
    fi
done

exit_code=0
for dir in $(go list ./...); do
    echo "golint $dir"
    result=$(golint $dir)
    if [ "" != "$result" ]; then
        echo $result
        exit_code=4
    fi
done

exit $exit_code
