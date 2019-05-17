#!/bin/bash

GOPATH="${GOPATH:-$HOME/go}"

go get github.com/cortesi/modd/cmd/modd
if [ ! -x ${GOPATH}/bin/modd ]
then
    go install github.com/cortesi/modd/cmd/modd
fi
