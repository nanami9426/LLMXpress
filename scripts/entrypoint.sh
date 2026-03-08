#!/usr/bin/env bash
set -euo pipefail

MODEL="${1:-Qwen/Qwen2.5-1.5B-Instruct}"
ARGS=(
  serve "$MODEL"
  --gpu-memory-utilization 0.75
  --max-model-len 4096
  --max-num-seqs 4
  --max-num-batched-tokens 1024
  --enable-force-include-usage
)

if command -v vllm >/dev/null 2>&1; then
  exec vllm "${ARGS[@]}"
fi

if command -v uv >/dev/null 2>&1; then
  echo "vllm not found on PATH, falling back to: uv run --with vllm" >&2
  exec uv run --no-project --with vllm -- vllm "${ARGS[@]}"
fi

echo "Neither 'vllm' nor 'uv' is available. Install vllm into your active Python environment, or install uv first." >&2
exit 127
