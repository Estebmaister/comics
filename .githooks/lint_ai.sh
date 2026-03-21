#!/bin/bash

printf "Running AI lint for security logic and performance...\n\n"

AI_URL="http://localhost:11434/api/generate"
AI_MODEL="phi4"

set -e

if git diff --cached --quiet; then
  printf "No staged changes detected.\n\n"
  exit 0
fi

tmp_prompt=$(mktemp) || {
  printf "⚠️ Warning: Could not create temp file for AI lint; skipping.\n\n" >&2
  exit 0
}
tmp_payload=$(mktemp) || {
  rm -f "$tmp_prompt"
  printf "⚠️ Warning: Could not create temp file for AI lint; skipping.\n\n" >&2
  exit 0
}
cleanup() { rm -f "$tmp_prompt" "$tmp_payload"; }
trap cleanup EXIT

{
  printf '%s\n\n' "Given the following git diff, identify any logic, performance, or security issues:"
  git diff --cached
} > "$tmp_prompt"

if ! jq -n \
  --arg model "$AI_MODEL" \
  --rawfile prompt "$tmp_prompt" \
  --argjson stream false \
  '{model: $model, prompt: $prompt, stream: $stream}' > "$tmp_payload"; then
  printf "⚠️ Warning: Failed to build AI lint request.\n\n" >&2
  exit 0
fi

if ! lint_response=$(curl -sS -f --connect-timeout 3 --max-time 120 -X POST "$AI_URL" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  --data-binary @"$tmp_payload" 2>/dev/null); then
  printf "⚠️ Warning: Failed to get a response from the AI service when linting (service unreachable or offline).\n\n" >&2
  exit 0
fi

if ! lint_text=$(printf "%s" "$lint_response" | tr '\n' '§' | jq -r '.response // empty' | tr '§' '\n' 2>/dev/null); then
  printf "⚠️ Warning: Failed to extract lint text from API response.\n\n" >&2
  exit 0
fi

printf "%s\n\n" "$lint_text"
