#!/bin/bash
ID='{ "identity" : { "account_number": "540155", "type": "User", "internal": { "org_id": "11009103" }, "user": { "is_org_admin": true, "username": "rh-ee-dagbay" } } }'
echo $ID | base64 | tr -d '\n'
