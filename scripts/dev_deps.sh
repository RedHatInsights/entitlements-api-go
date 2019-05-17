#!/bin/bash

go get github.com/cortesi/modd/cmd/modd
if [ ! -x ~/go/bin/modd ]
then
    go install github.com/cortesi/modd/cmd/modd
fi
