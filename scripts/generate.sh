#!/usr/bin/env bash
#
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
APISPEC_DIR="$SCRIPT_DIR/../apispec"
OPENAPI_DIR="$SCRIPT_DIR/../openapi"

oapi-codegen -include-tags seats -generate "types,chi-server" -package openapi $APISPEC_DIR/api.spec.json > $OPENAPI_DIR/openapi.go
