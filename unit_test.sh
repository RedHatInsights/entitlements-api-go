#!/bin/bash

make test-all

if [ $? != 0 ]; then
    exit 1
fi