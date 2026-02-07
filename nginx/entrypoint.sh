#!/bin/sh
set -e
# Substitute DOMAIN in nginx config
export DOMAIN="${DOMAIN:-localhost}"
envsubst '${DOMAIN}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
# Wait for Certbot to create certs (when running compose up after initial certbot run)
CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
while [ ! -f "${CERT_DIR}/fullchain.pem" ] || [ ! -f "${CERT_DIR}/privkey.pem" ]; do
    echo "Waiting for certificates at ${CERT_DIR}..."
    sleep 5
done
exec nginx -g 'daemon off;'
