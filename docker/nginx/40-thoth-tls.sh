#!/bin/sh
# Runs from the nginx image's /docker-entrypoint.d before nginx starts.
# When NGINX_TLS=1, swap the plain-HTTP site for the TLS one.
set -eu

if [ "${NGINX_TLS:-0}" != "1" ]; then
    exit 0
fi

: "${SERVER_NAME:?SERVER_NAME is required when NGINX_TLS=1}"
: "${TLS_CERT_NAME:?TLS_CERT_NAME is required when NGINX_TLS=1}"

envsubst '${SERVER_NAME} ${TLS_CERT_NAME}' \
    < /etc/nginx/thoth/tls.conf.template \
    > /etc/nginx/conf.d/default.conf

echo "40-thoth-tls.sh: TLS enabled for ${SERVER_NAME} (cert: ${TLS_CERT_NAME})"
