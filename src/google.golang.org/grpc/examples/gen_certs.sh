#!/bin/bash

echo -e "\n==== Gen server certs... ===="
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
-keyout ./certs/backend.key \
-out ./certs/backend.crt \
-subj "/CN=backend.local/O=backend.local"


echo -e "\n==== Gen client certs... ===="
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
-keyout ./certs/frontend.key \
-out ./certs/frontend.crt \
-subj "/CN=frontend.local/O=frontend.local"
