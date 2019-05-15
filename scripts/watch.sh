#!/bin/bash

if [[ $1 ]]
then
    source $1
else
    source ./config/development.iphands.sh
fi

GOPATH=~/go ~/go/bin/watcher -run github.com/RedHatInsights/entitlements-api-go
