#!/bin/sh
# migrate.sh — switch an existing upstream Cloudreve docker-compose deployment
# to this fork. All data (volumes, database, env config) is preserved; only the
# container image changes. Usage: sh migrate.sh [compose-project-dir]

set -eu

DIR="${1:-.}"
IMAGE="ghcr.io/dvorinka/cloudreve:latest"

for f in docker-compose.yml docker-compose.yaml compose.yml compose.yaml; do
    if [ -f "$DIR/$f" ]; then COMPOSE_FILE="$DIR/$f"; break; fi
done

if [ -z "${COMPOSE_FILE:-}" ]; then
    echo "No compose file found in $DIR" >&2
    exit 1
fi

if ! grep -q "cloudreve/cloudreve" "$COMPOSE_FILE"; then
    echo "No upstream 'cloudreve/cloudreve' image found in $COMPOSE_FILE — nothing to migrate." >&2
    exit 1
fi

cp "$COMPOSE_FILE" "$COMPOSE_FILE.bak"
sed -i "s|image: *cloudreve/cloudreve:[^ ]*|image: $IMAGE|" "$COMPOSE_FILE"
echo "Updated $COMPOSE_FILE (backup: $COMPOSE_FILE.bak)"

cd "$DIR"
if docker compose version >/dev/null 2>&1; then
    docker compose pull && docker compose up -d
elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose pull && docker-compose up -d
else
    echo "docker compose not found — install it, then run: docker compose pull && docker compose up -d" >&2
    exit 1
fi

echo "Migration complete. Cloudreve is on port 5212; existing accounts, files, and settings carry over."
