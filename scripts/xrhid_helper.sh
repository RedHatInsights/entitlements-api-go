#!/bin/bash

# ID='{ "account_number": "540155", "internal": { "org_id": "6340056" } }'
ID='{ "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } }'

curl -v \
     -H "x-rh-identity: `echo -n $ID | base64 -w0`" \
     http://localhost:3000/api/entitlements/v1/services
