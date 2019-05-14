#!/bin/bash

if [[ $1 ]]
then
    source $1
else
    source ./config/development.iphands.sh
fi

GOPATH=~/go ~/go/bin/watcher -run cloud.redhat.com/entitlements
