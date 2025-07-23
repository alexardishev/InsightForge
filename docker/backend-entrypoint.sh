#!/bin/sh
set -e

# Run database migrations for the system DB
if [ -n "$MIGRATOR_STORAGE_PATH" ]; then
    /usr/local/bin/migrator -storage-path "$MIGRATOR_STORAGE_PATH" -migrations-path /app/migrations
fi

exec /usr/local/bin/server
