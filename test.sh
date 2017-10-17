#!/usr/bin/env bash

go test -race -v ./...
ret=$?; if [[ $ret -ne 0 && $ret -ne 1 ]]; then
    echo "Test failed, exit $ret"
    exit $ret
fi

echo "mode: atomic" > coverage.txt

function coverage() {
    go test $1 -race -coverprofile=tmp.txt -covermode=atomic
    ret=$?; if [[ $ret -eq 0 ]]; then
        cat tmp.txt >> coverage.txt
        rm -f tmp.txt
    fi
}

coverage github.com/ossrs/go-oryx/kernel

