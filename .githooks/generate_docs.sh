# generate_docs.sh
#!/bin/bash

printf "Generating changelog...\n\n"

AI_URL="http://localhost:11434/api/generate"
AI_MODEL="phi4"
CHANGELOG_DIR="changelog"

# Ensure the script exits immediately if a command exits with a non-zero status
set -e

# Get the diff between the last commit and current state
ch_diff=$(git diff HEAD~1)

# Build prompt
ch_prompt="Generate a concise changelog entry, 
which output will be directly added to the changelog .md file, 
for the following diff:\n\n$ch_diff"

# Prepare the JSON payload
json_changelog_payload=$(jq -n \
  --arg model "$AI_MODEL" \
  --arg prompt "$ch_prompt" \
  --argjson stream false \
  '{model: $model, prompt: $prompt, stream: $stream}')

# Send the request to the AI API
changelog_response=$(curl -s -X POST $AI_URL \
  -H "Content-Type: application/json" \
  -d "$json_changelog_payload") || {
    printf "❌ Error: Failed to get a response from the AI service when generating changelog.\n\n" >&2
    exit 1
}

# Extract the response text with better error checking
changelog_entry=$(printf "%s" "$changelog_response" | tr '\n' '§' | jq -r '.response' | tr '§' '\n') || {
    printf "❌ Error: Failed to extract changelog text from API response.\n\n" >&2
    exit 1
}

# Define the changelog file path
changelog_file="$CHANGELOG_DIR/$(date +%Y-%m-%d_%H-%M-%S).md"

# Create the changelog directory if it doesn't exist
mkdir -p "$CHANGELOG_DIR"

# Write the changelog entry to the file
echo "$changelog_entry" > "$changelog_file"

printf "✅ Changelog generated: $changelog_file\n\n"

# Stage the changelog file
git add "$changelog_file"