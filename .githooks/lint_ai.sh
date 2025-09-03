#!/bin/bash

printf "Running AI lint for security logic and performance...\n\n"

AI_URL="http://localhost:11434/api/generate"
AI_MODEL="phi4"

# Ensure the script exits immediately if a command exits with a non-zero status
set -e

# Retrieve the staged git diff
lint_diff=$(git diff --cached)

# Check if there are any staged changes
if [ -z "$lint_diff" ]; then
  printf "No staged changes detected.\n\n"
  exit 0
fi

# Build prompt
lint_prompt="Given the following git diff, identify any logic, performance, or security issues:\n\n$lint_diff"

# Prepare the JSON payload
json_lint_payload=$(jq -n \
  --arg model "$AI_MODEL" \
  --arg prompt "$lint_prompt" \
  --argjson stream false \
  '{model: $model, prompt: $prompt, stream: $stream}')

# Send the request to the AI API
lint_response=$(curl -s -X POST $AI_URL \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d "$json_lint_payload") || {
    printf "⚠️ Warning: Failed to get a response from the AI service when linting.\n\n" >&2
    exit 0
}

# Extract and display the response
lint_response=$(printf "%s" "$lint_response" | tr '\n' '§' | jq -r '.response' | tr '§' '\n') || {
    printf "❌ Error: Failed to extract lint text from API response.\n\n" >&2
    exit 1
}

printf "%s\n\n" "$lint_response"