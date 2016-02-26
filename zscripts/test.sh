#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/mdb
go test -cover -coverprofile=coverage.out

if [ "$1" == "html" ]; then
    go tool cover -html=coverage.out
fi
