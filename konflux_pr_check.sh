#!/bin/bash

echo "go version" 

go version 

make generate
go mod download
go get ./... 

make test-all