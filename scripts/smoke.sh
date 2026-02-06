#!/usr/bin/env bash
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:5000}"
JWT="${JWT:-}"
MODEL="${MODEL:-Qwen/Qwen2.5-1.5B-Instruct}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1"; exit 1; }; }
need curl
JQ_OK=1
command -v jq >/dev/null 2>&1 || JQ_OK=0

echo "== Smoke: gateway=$GATEWAY_URL model=$MODEL =="

echo "-- 1) /healthz"
curl -fsS "$GATEWAY_URL/healthz" >/dev/null
echo "OK"

echo "-- 2) /v1/models without JWT should be 401"
code="$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY_URL/v1/models" || true)"
if [[ "$code" != "401" ]]; then
  echo "Expected 401, got $code"
  exit 1
fi
echo "OK"

if [[ -z "$JWT" ]]; then
  echo "JWT env is empty. Export JWT then re-run."
  echo "Example: export JWT='xxx.yyy.zzz'"
  exit 1
fi

echo "-- 3) /v1/models with JWT should be 200 and contain data[]"
resp_models="$(curl -fsS "$GATEWAY_URL/v1/models" -H "Authorization: Bearer $JWT")"
if [[ "$JQ_OK" == "1" ]]; then
  echo "$resp_models" | jq -e '.data[0].id' >/dev/null
  echo "Model[0] = $(echo "$resp_models" | jq -r '.data[0].id')"
else
  echo "$resp_models" | head -c 200; echo
fi
echo "OK"

echo "-- 4) /v1/chat/completions basic non-stream"
req="$(cat <<JSON
{
  "model":"$MODEL",
  "messages":[
    {"role":"system","content":"You are a helpful assistant."},
    {"role":"user","content":"Say hello in one sentence."}
  ],
  "temperature":0.0,
  "max_tokens":64,
  "stream":false
}
JSON
)"
resp_chat="$(curl -fsS "$GATEWAY_URL/v1/chat/completions" \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d "$req")"

if [[ "$JQ_OK" == "1" ]]; then
  echo "$resp_chat" | jq -e '.choices[0].message.content' >/dev/null
  echo "Assistant: $(echo "$resp_chat" | jq -r '.choices[0].message.content' | head -c 120)"
else
  echo "$resp_chat" | head -c 200; echo
fi

echo "== Smoke PASS =="
