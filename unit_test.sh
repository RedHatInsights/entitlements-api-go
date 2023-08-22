#!/bin/bash

ACG_CONFIG="$(pwd)/cdappconfig.json" make test-all

if [ $? != 0 ]; then
    exit 1
fi