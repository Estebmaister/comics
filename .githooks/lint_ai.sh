#!/usr/bin/env sh
# Advisory AI review of staged diff; never fails the commit.

ROOT=$(git rev-parse --show-toplevel)
# shellcheck source=/dev/null
. "$ROOT/.githooks/common.sh"

comics_cd_root || exit 0

if [ "${COMICS_AI_LINT:-}" != "1" ]; then
  exit 0
fi

if ! comics_ai_hooks_enabled; then
  printf "⚠️  COMICS_AI_LINT=1 but COMICS_HOOKS_AI is unset or jq/curl missing; skipping.\n\n" >&2
  exit 0
fi

if git diff --cached --quiet; then
  exit 0
fi

printf "Running AI lint (advisory)...\n\n"

tmp_prompt=$(mktemp) || exit 0
trap 'rm -f "$tmp_prompt"' EXIT

{
  printf '%s\n\n' "Review this staged git diff for logic, performance, or security issues. Be brief; say if nothing stands out."
  comics_staged_diff
} > "$tmp_prompt" || exit 0

lint_text=$(comics_ollama_generate "$tmp_prompt") || {
  printf "⚠️  AI lint: Ollama unreachable or failed; skipping.\n\n" >&2
  exit 0
}

[ -n "$lint_text" ] && printf '%s\n\n' "$lint_text"
exit 0
