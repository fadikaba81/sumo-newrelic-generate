#!/usr/bin/env bash

BASE_URL="${BASE_URL:-https://localhost:8443}"

for i in $(seq 1 100); do
  START=$(date -u -d "$i minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
  END=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  code=$(curl -sk -o /dev/null -w "%{http_code}" --get "$BASE_URL/order" \
    --data-urlencode "start=$START" \
    --data-urlencode "end=$END" || echo "000")

  echo "Request $i (HTTP $code, window $START to $END)"
  sleep 0.5
done