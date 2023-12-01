#!/usr/bin/env bash
set -euo pipefail

echo 'Generating selfsign root CA'
openssl req -x509 -nodes -new -sha256 -days 3650 -newkey rsa:2048 -keyout root-ca.key -out root-ca.pem -subj "/C=DE/ST=NRW/O=trivago/CN=localhost"
openssl x509 -outform pem -in root-ca.pem -out root-ca.crt

echo 'Generating server cert'
openssl req -new -nodes -newkey rsa:2048 -subj "/C=DE/ST=NRW/L=Duesseldorf/O=trivago/CN=localhost" -keyout tls.key -out localhost.csr
openssl x509 -req -sha256 -days 3650 -in localhost.csr -CA root-ca.pem -CAkey root-ca.key -CAcreateserial -extfile domains.ext -out tls.cert
