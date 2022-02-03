#!/usr/bin/env sh
set -eu

envsubst '${DLE_HOST} ${DLE_PORT}' < /etc/nginx/conf.d/ce.conf.template > /etc/nginx/conf.d/ce.conf

exec "$@"
