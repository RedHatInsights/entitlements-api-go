#!/bin/bash

echo "go version" 

go version 

go mod download
go get ./... 

make test-all