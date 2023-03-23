#!/bin/bash
ID='{ "identity" : { "account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }'
echo $ID | base64 | tr -d '\n'
