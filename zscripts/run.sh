#!/bin/sh
# Worker pool test for database access control.
cd $GOPATH/src/github.com/jimmy-go/mdb/examples
go build -race -o $GOBIN/mdb && \
$GOBIN/mdb -tasks=100 -max-workers=3 -max-queue=10 \
-host=localhost -port=10000 -database=pompitos -username=8Y59e0DUcf -password=yi7Sry1KEb
#-host=192.168.15.64 -port=10001 -database=alerta_mobil -username=stress -password=stress
