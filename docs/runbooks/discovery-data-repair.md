# Discovery data repair runbook

## Read-only checks

1. Stop automatic scrapes if you need a stable snapshot (optional for audit-only).
2. `make db-audit-discovery` — summary counts and logical fingerprint.
3. `make db-audit-discovery-live` — same audit with file digest guard (`AUDIT_DISCOVERY=1`).

## Export and review

1. `make db-backup BACKUP_FILE=db/backups/comics_pre_repair.db`
2. `make db-export-discovery-plan DISCOVERY_PLAN=db/recovery/discovery-repair.json`
3. Review JSON groups; set `review.approved`, `review.decision` (`restore_keys` or `merge`), `reviewer`, and `reviewed_at` (RFC3339).
4. `make db-validate-discovery-plan DISCOVERY_PLAN=db/recovery/discovery-repair.json`

## Apply (offline)

1. Stop the API and scraper.
2. `make db-apply-discovery-plan CONFIRM=reviewed DISCOVERY_PLAN=... DISCOVERY_BACKUP=... DISCOVERY_GROUP=keyless:123`
   - Or `DISCOVERY_ALL_APPROVED=1` to apply every approved group in one transaction.
3. Re-run `make db-audit-discovery` and compare keyless/conflict counts.

## Observability

- Ambiguous scrape matches increment `comics_discovery_scrape_registration_total{result="ambiguous"}` and do not emit warn-level registration logs (structured logging is a later observability task).

## Rollback

Restore the verified backup: `make db-restore BACKUP=db/backups/comics_pre_repair.db` (stop API first).
