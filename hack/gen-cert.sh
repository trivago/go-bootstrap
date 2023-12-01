#!/usr/bin/env bash
set -euo pipefail

openssl req -new -subj "/C=US/ST=Utah/CN=localhost" -newkey rsa:2048 -nodes -keyout tls.key -out localhost.csr
openssl x509 -req -days 3650 -in localhost.csr -signkey tls.key -out tls.cert
