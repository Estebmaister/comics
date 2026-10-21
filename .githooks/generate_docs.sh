#!/usr/bin/env sh
# Optional AI changelog file (not staged unless COMICS_AI_CHANGELOG_STAGE=1).

set -e

ROOT=$(git rev-parse --show-toplevel)
# shellcheck source=/dev/null
. "$ROOT/.githooks/common.sh"

comics_cd_root || exit 0

if [ "${COMICS_AI_CHANGELOG:-}" != "1" ]; then
  exit 0
fi

if ! comics_ai_hooks_enabled; then
  printf "⚠️  COMICS_AI_CHANGELOG=1 but COMICS_HOOKS_AI is unset or jq/curl missing; skipping changelog.\n\n" >&2
  exit 0
fi

if git diff --cached --quiet; then
  exit 0
fi

printf "Generating changelog (staged diff)...\n\n"

CHANGELOG_DIR="changelog"
tmp_prompt=$(mktemp) || exit 0
trap 'rm -f "$tmp_prompt"' EXIT

{
  printf '%s\n\n' "Generate a concise changelog entry for the following staged diff. Use markdown bullets."
  comics_staged_diff
} > "$tmp_prompt" || exit 0

changelog_entry=$(comics_ollama_generate "$tmp_prompt") || {
  printf "⚠️  Changelog: Ollama unreachable or failed; skipping.\n\n" >&2
  exit 0
}

[ -n "$changelog_entry" ] || exit 0

changelog_file="$CHANGELOG_DIR/$(date +%Y-%m-%d_%H-%M-%S).md"
mkdir -p "$CHANGELOG_DIR"
printf '%s\n' "$changelog_entry" > "$changelog_file"
printf "✅ Changelog written: %s\n\n" "$changelog_file"

if [ "${COMICS_AI_CHANGELOG_STAGE:-}" = "1" ]; then
  git add "$changelog_file"
  printf "   (staged — unset COMICS_AI_CHANGELOG_STAGE to write without staging)\n\n"
fi

exit 0
