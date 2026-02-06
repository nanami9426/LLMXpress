#!/usr/bin/env bash
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:5000}"
JWT="${JWT:-}"
MODEL="${MODEL:-Qwen/Qwen2.5-1.5B-Instruct}"
N="${1:-50}"          # 总请求数
C="${2:-5}"           # 并发
OUTDIR="${OUTDIR:-artifacts}"
TS="$(date +%Y%m%d_%H%M%S)"
OUT="$OUTDIR/bench_${TS}_n${N}_c${C}.log"

mkdir -p "$OUTDIR"

if [[ -z "$JWT" ]]; then
  echo "JWT env is empty. export JWT then run."
  exit 1
fi

payload="$(cat <<JSON
{
  "model":"$MODEL",
  "messages":[{"role":"user","content":"Write a short 10-word sentence about cats."}],
  "temperature":0.0,
  "max_tokens":32,
  "stream":false
}
JSON
)"

echo "== Bench: n=$N c=$C gateway=$GATEWAY_URL model=$MODEL ==" | tee "$OUT"

# 单次请求函数：输出 "code time_total"
one() {
  curl -sS -o /dev/null \
    -w "%{http_code} %{time_total}\n" \
    "$GATEWAY_URL/v1/chat/completions" \
    -H "Authorization: Bearer $JWT" \
    -H "Content-Type: application/json" \
    -d "$payload" || echo "000 0"
}
export -f one
export GATEWAY_URL JWT payload

# 并发发 N 次
seq 1 "$N" | xargs -n1 -P "$C" -I{} bash -c 'one' | tee -a "$OUT" >/tmp/bench_times_${TS}.txt

# 统计
total="$(wc -l < /tmp/bench_times_${TS}.txt | tr -d ' ')"
ok="$(awk '$1 ~ /^2/ {c++} END{print c+0}' /tmp/bench_times_${TS}.txt)"
fail="$((total - ok))"

# 提取耗时（只统计 2xx 的 time_total）
awk '$1 ~ /^2/ {print $2}' /tmp/bench_times_${TS}.txt | sort -n > /tmp/bench_lat_${TS}.txt
okn="$(wc -l < /tmp/bench_lat_${TS}.txt | tr -d ' ')"

pct() {
  local p="$1"
  if [[ "$okn" -eq 0 ]]; then echo "NaN"; return; fi
  # rank = ceil(p/100 * okn)
  local rank
  rank="$(awk -v p="$p" -v n="$okn" 'BEGIN{r=int((p/100.0)*n); if((p/100.0)*n>r) r=r+1; if(r<1) r=1; if(r>n) r=n; print r}')"
  awk -v r="$rank" 'NR==r{print $1}' /tmp/bench_lat_${TS}.txt
}

avg="$(awk '{s+=$1} END{if(NR>0) printf "%.6f", s/NR; else print "NaN"}' /tmp/bench_lat_${TS}.txt)"
p50="$(pct 50)"
p95="$(pct 95)"
p99="$(pct 99)"
minv="$(awk 'NR==1{print $1}' /tmp/bench_lat_${TS}.txt 2>/dev/null || echo NaN)"
maxv="$(awk 'END{print $1}' /tmp/bench_lat_${TS}.txt 2>/dev/null || echo NaN)"

echo "" | tee -a "$OUT"
echo "== Result ==" | tee -a "$OUT"
echo "total=$total ok=$ok fail=$fail success_rate=$(awk -v ok="$ok" -v t="$total" 'BEGIN{if(t>0) printf "%.2f%%", 100*ok/t; else print "NaN"}')" | tee -a "$OUT"
echo "latency_seconds: min=$minv avg=$avg p50=$p50 p95=$p95 p99=$p99 max=$maxv" | tee -a "$OUT"
echo "raw_log=$OUT" | tee -a "$OUT"

rm -f /tmp/bench_times_${TS}.txt /tmp/bench_lat_${TS}.txt
