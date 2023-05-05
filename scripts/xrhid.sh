#!/bin/bash
ID='{ "identity" : { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" }, "user": { "is_org_admin": true } } }'
echo $ID | base64 | tr -d '\n'
