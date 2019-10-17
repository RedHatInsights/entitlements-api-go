#!/bin/bash

if test -f "$1"
then
    source $1
else
    echo "You must pass in a config file to be sourced!"
    exit 1
fi

export ENT_CA_PATH=./resources/ca.crt

GOPATH="${GOPATH:-$HOME/go}"
$GOPATH/bin/modd
