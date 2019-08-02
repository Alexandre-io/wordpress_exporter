#!/bin/bash
set -e

if [[ -n "${WORDPRESS_DB_HOST}" ]]; then
    exec "$@" -host="${WORDPRESS_DB_HOST}" -port="${WORDPRESS_DB_PORT}" -user="${WORDPRESS_DB_USER}" -db="${WORDPRESS_DB_NAME}" -tableprefix="${WORDPRESS_TABLE_PREFIX}" -pass="${WORDPRESS_DB_PASSWORD}"
else
    exec "$@"
fi
