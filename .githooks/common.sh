# Shared helpers for comics git hooks. Source from repo root:
#   . "$(git rev-parse --show-toplevel)/.githooks/common.sh"

comics_repo_root() {
  git rev-parse --show-toplevel 2>/dev/null
}

comics_cd_root() {
  root=$(comics_repo_root) || return 1
  cd "$root" || return 1
}

# Skip all optional / AI hook work (server.pid prompt is still relaxed when non-interactive).
comics_skip_hooks() {
  [ -n "${COMICS_SKIP_HOOKS:-}" ]
}

# Master switch for Ollama-backed hooks (lint, changelog, commit-msg assist).
comics_ai_hooks_enabled() {
  if comics_skip_hooks; then
    return 1
  fi
  [ -n "${COMICS_HOOKS_AI:-}" ] || return 1
  command -v jq >/dev/null 2>&1 || return 1
  command -v curl >/dev/null 2>&1 || return 1
  return 0
}

COMICS_AI_URL="${COMICS_AI_URL:-http://localhost:11434/api/generate}"
COMICS_AI_MODEL="${COMICS_AI_MODEL:-phi4}"

# Staged textual diff for prompts; excludes large binaries and generated protobuf.
comics_staged_diff() {
  git diff --cached --no-color -- \
    ':!*.db' ':!*.sqlite' ':!*.sqlite3' \
    ':!*.png' ':!*.jpg' ':!*.jpeg' ':!*.gif' ':!*.webp' \
    ':!*.woff' ':!*.woff2' ':!*.pdf' \
    ':!go_server/pkg/pb/*' \
    ':!src/pb/*' \
    ':!src/frontend/pb/*' 2>/dev/null | head -c "${COMICS_AI_DIFF_MAX_BYTES:-120000}"
}

# Call Ollama generate API; prints model text to stdout or returns non-zero.
comics_ollama_generate() {
  prompt_file=$1
  if [ ! -s "$prompt_file" ]; then
    return 1
  fi
  payload=$(mktemp) || return 1
  # shellcheck disable=SC2064
  trap 'rm -f "$payload"' EXIT
  if ! jq -n \
    --arg model "$COMICS_AI_MODEL" \
    --rawfile prompt "$prompt_file" \
    --argjson stream false \
    '{model: $model, prompt: $prompt, stream: $stream}' > "$payload"; then
    return 1
  fi
  response=$(curl -sS -f --connect-timeout 3 --max-time "${COMICS_AI_TIMEOUT:-120}" \
    -X POST "$COMICS_AI_URL" \
    -H "Content-Type: application/json" \
    --data-binary @"$payload" 2>/dev/null) || return 1
  printf '%s' "$response" | jq -r '.response // empty' 2>/dev/null
}

# True if commit message file has a non-empty, non-comment subject line.
comics_commit_msg_has_subject() {
  file=$1
  [ -f "$file" ] || return 1
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      ''|'#'*) continue ;;
      *) return 0 ;;
    esac
  done < "$file"
  return 1
}
