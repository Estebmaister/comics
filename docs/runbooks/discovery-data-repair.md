# Discovery data repair runbook

## Gates (read before any write)

- Stop the API and scrape scheduler before **backup** or **apply**.
- `db-audit-discovery`, `db-export-discovery-plan`, and `db-validate-discovery-plan` are **read-only**.
- Inspect every conflict: titles, types, chapters, tracking, publishers, and metadata.
- Never approve merges across a **novel/series** boundary.
- Far chapters (`spread > 50` or `max >= 30` and `ratio < 0.50`) require human review regardless of title similarity.
- Every approved group needs `reviewer`, RFC3339 `reviewed_at`, `decision`, and notes; merges also need `canonical_id` and complete `merge_ids`.
- Rehearse a **canary group** on a copy (`db/recovery/discovery-rehearsal.db`) before touching production.
- Use `DISCOVERY_ALL_APPROVED=1` only when every approved group has full review metadata.
- Roll back with `make db-restore` if post-apply audit or smoke checks fail.
- Every restore requires the API/scheduler stopped and `CONFIRM_OFFLINE=yes`.
- Repair SQLite first, then run the guarded PostgreSQL import; do not repair the two catalogs independently.

## Read-only checks

1. `make db-audit-discovery` — summary counts and logical fingerprint.
2. `make db-audit-discovery-live` — same audit with file digest guard (`AUDIT_DISCOVERY=1`).

## Export and review

1. `make db-backup BACKUP_FILE=db/backups/comics_pre_repair.db`
2. `make db-export-discovery-plan DISCOVERY_PLAN=db/recovery/discovery-repair.json`
3. Edit the plan JSON; never auto-approve in tooling.
4. `make db-validate-discovery-plan DISCOVERY_PLAN=db/recovery/discovery-repair.json`

## Apply (offline)

1. Stop the API and scraper.
2. `make db-apply-discovery-plan CONFIRM=reviewed DISCOVERY_PLAN=... DISCOVERY_BACKUP=... DISCOVERY_GROUP=keyless:123`
   - Or `DISCOVERY_ALL_APPROVED=1` to apply every approved group in **one transaction** (any failure rolls back all).
3. `make db-audit-discovery` and compare keyless/conflict counts.

## Rehearsal copy (recommended)

```bash
make db-backup DB_FILE=db/comics.db BACKUP_FILE=db/backups/discovery_rehearsal_source.db
cp db/backups/discovery_rehearsal_source.db db/recovery/discovery-rehearsal.db
make db-backup DB_FILE=db/recovery/discovery-rehearsal.db BACKUP_FILE=db/backups/discovery_rehearsal_before_apply.db
make db-export-discovery-plan DB_FILE=db/recovery/discovery-rehearsal.db DISCOVERY_PLAN=db/recovery/discovery-rehearsal-plan.json
# review + jq edit → discovery-rehearsal-plan.reviewed.json
make db-validate-discovery-plan DB_FILE=db/recovery/discovery-rehearsal.db DISCOVERY_PLAN=db/recovery/discovery-rehearsal-plan.reviewed.json
make db-apply-discovery-plan DB_FILE=db/recovery/discovery-rehearsal.db DISCOVERY_PLAN=db/recovery/discovery-rehearsal-plan.reviewed.json DISCOVERY_BACKUP=db/backups/discovery_rehearsal_before_apply.db DISCOVERY_GROUP=keyless:5980 CONFIRM=reviewed
```

## Observability

- **Metrics:** `comics_discovery_scrape_registration_total{result="ambiguous"}` and `comics_discovery_resolution_total{outcome="ambiguous"}`.
- **Logs:** ambiguous scrapes emit **info**-level structured lines (`scrape register ambiguous match` with `candidate_ids`); other registration failures stay at **warn**. Broader correlation and dashboards live in the observability workstream.

## Rollback

```bash
make db-restore BACKUP=db/backups/comics_pre_repair.db CONFIRM_OFFLINE=yes
make db-check
make db-audit-discovery
```
