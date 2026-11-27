#!/usr/bin/env bash
set -euo pipefail

URL="https://localhost:8443/"

for i in $(seq 1 1000); do
  curl -sk -o /dev/null -w "%{http_code}\n" "$URL" \
    -H "X-Correlation-ID: OENI7V9p1kqe"
  curl -sk -o /dev/null -w "%{http_code}\n" "$URL"
  sleep 0.2
done