#!/usr/bin/env bash
# Boots an image as a non-root container with the host mounted, the way
# people run dashdot, and waits for every integration route to answer.
set -euo pipefail

image="$1"
name="dash-comma-smoke-$$"

docker run -d --name "$name" -p 127.0.0.1:3001:3001 -v /:/mnt/host:ro \
  -e DASHDOT_WIDGET_LIST=os,cpu,storage,ram,network,gpu "$image" >/dev/null
trap 'docker rm -f "$name" >/dev/null' EXIT

for _ in $(seq 1 30); do
  if curl -fs -o /dev/null http://127.0.0.1:3001/info; then
    for route in / config info load/cpu load/ram load/storage load/network load/gpu; do
      curl -fsS -o /dev/null "http://127.0.0.1:3001/${route#/}" || { echo "smoke test failed: /${route#/}" >&2; exit 1; }
    done
    # The compose healthcheck dashdot users have: curl must be in the image.
    docker exec "$name" curl -fsS -o /dev/null http://localhost:3001/
    docker exec "$name" dash-comma healthcheck
    echo "smoke test passed: $image answers on every route"
    exit 0
  fi
  sleep 1
done

echo "smoke test failed: $image did not answer on /info within 30s" >&2
docker logs --tail 50 "$name" >&2
exit 1
