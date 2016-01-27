#!/bin/sh
# Worker pool test for database access control.
cd $GOPATH/src/github.com/jimmy-go/mdb/examples
go clean
go build -o mdb && \
./mdb -tasks=2000 -max-workers=3 -max-queue=5 -host=localhost -database=pompitos -username=8Y59e0DUcf -password=yi7Sry1KEb
