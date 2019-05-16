#!/bin/bash

if test -f "$1"
then
    source $1
else
    echo "You must pass in a config file to be sourced!"
    exit 1
fi

GOPATH=~/go ~/go/bin/modd
