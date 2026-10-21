# Git hooks (`core.hooksPath=.githooks`)

Hooks run from the repo root. Configure once:

```bash
make setup-hooks
```

## Always on

- **pre-commit**: blocks `server.pid` from being committed; warns if `server.pid` exists (interactive prompt only on a TTY).

## Opt-in AI (Ollama)

AI hooks are **off by default** so `git commit -m "..."` stays fast and your message is never overwritten.

| Variable | Effect |
|----------|--------|
| `COMICS_HOOKS_AI=1` | Enable Ollama calls (needs `jq`, `curl`, phi4 or `COMICS_AI_MODEL` on `localhost:11434`) |
| `COMICS_SKIP_HOOKS=1` | Skip optional AI work entirely |
| `COMICS_AI_LINT=1` | Advisory security/performance note on staged diff (pre-commit) |
| `COMICS_AI_CHANGELOG=1` | Write `changelog/<timestamp>.md` from **staged** diff (not auto-staged) |
| `COMICS_AI_CHANGELOG_STAGE=1` | Also `git add` the new changelog file |
| `COMICS_AI_COMMIT_MSG=1` | On interactive commits only, append an AI suggestion as `#` comments in the editor |

`prepare-commit-msg` **never** runs for `git commit -m` (`source=message`). `--no-verify` skips **pre-commit** only; it does not disable `prepare-commit-msg`.

### One-shot “old behavior” (slow)

```bash
export COMICS_HOOKS_AI=1
export COMICS_AI_LINT=1
export COMICS_AI_CHANGELOG=1
export COMICS_AI_CHANGELOG_STAGE=1
export COMICS_AI_COMMIT_MSG=1
git commit
```

### Overrides

- `COMICS_AI_URL` — default `http://localhost:11434/api/generate`
- `COMICS_AI_MODEL` — default `phi4`
- `COMICS_AI_DIFF_MAX_BYTES` — cap diff size sent to the model (default `120000`)

Staged diffs exclude binaries (`.db`, images, etc.) and generated `pb/` trees.
