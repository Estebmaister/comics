# Discovery and Data Integrity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore discovery keys for the 346 active keyless comics through a read-only, human-reviewed repair workflow, then prevent new keyless or ambiguously matched records on every catalog write.

**Architecture:** Keep identity normalization in `internal/identity`, resolution policy and bounded metrics in `internal/discovery`, and SQL ownership enforcement in `internal/repo/comics`. Audit/export opens SQLite in query-only mode; exported plans never approve a merge. Offline maintenance validates the reviewed plan and a matching backup before applying either one canary group or all explicitly approved groups in one transaction.

**Tech Stack:** Go 1.27.3, `database/sql`, modernc SQLite, pgx PostgreSQL, Prometheus client, Gin, JSON repair plans, GNU Make.

## Global Constraints

- SQLite remains the default catalog store; PostgreSQL remains optional.
- MongoDB remains the user/auth store and is not changed by this workstream.
- Do not introduce Redis, another service, or a microservice split.
- Every active comic must have at least one valid discovery key after reviewed repair.
- One `(scope, title_key)` has exactly one active owner.
- Discovery-key conflicts are typed errors; an insert conflict is never discarded silently.
- Comic-row and discovery-key changes commit or roll back together.
- Create, update, merge, import, and scrape paths all use `identity.NormalizeTitleVariants`.
- Update duplicate checks exclude the comic being updated.
- Deleted comics hold no discovery keys and cannot block an active owner.
- Resolver ties or near-ties are ambiguous; tracking state, chapter count, and lower ID never break an identity tie.
- Audit and export are database-read-only. Export writes only the requested JSON plan.
- Every exported repair group starts with `approved: false`; there is no automatic merge approval.
- SQLite repair runs offline with one API replica and one scheduler stopped.
- Runtime SQLite/PostgreSQL behavior must pass the same discovery repository contract.
- Workstream 3 owns the SQLite foreign-key table rebuild and migration-location cleanup. This plan adds only the transaction wrapping needed to keep comic/key writes atomic now.

---

## Verified Baseline and Scope Boundary

Read-only inspection on 2026-10-26 established:

```text
PRAGMA integrity_check: ok
comics: total=31309 active=31308
comic_title_keys: rows=31892 active_owners=30962
keyless active comics: 346
keyless records with an active key owner: 341
keyless records made keyable by the Unicode-only fallback: 4
keyless records with no usable title: 1 (comic 4561)
deleted key rows: 0
orphan key rows: 0
```

The Unicode change in Task 1 is deliberately a fallback. Existing ASCII-derived keys remain byte-for-byte unchanged; only a title that currently produces an empty ASCII key uses Unicode letters and numbers. The four affected live records are 5980, 8272, 14023, and 18531. Comic 4561 has an empty title and remains blocked until an operator supplies a title or explicitly deletes it.

This workstream does not decide which of the 341 conflicting pairs are the same series. It supplies exact evidence, defaults every group to review, and applies only recorded human decisions.

## Dependencies and Sequencing

- Implement after Workstream 1's route split, or rebase Task 5's 409 mapping onto it before commit. The owner-ID response is valid only on admin mutation routes.
- Complete Tasks 1-8 before Workstream 3 rebuilds the SQLite title-key table. Workstream 3 must preserve the conflict types and repository contract established here.
- Complete the reviewed SQLite repair before using `go-import-postgres`; the importer intentionally rejects a keyless or conflicting source.
- Workstream 9 may consolidate collector registration, but it must preserve the metric names and bounded labels introduced in Task 4.
- Code and tests can land before live repair. Do not run Task 9's production sequence until a fresh backup, human-reviewed plan, and successful rehearsal all exist.

## File and Responsibility Map

Files created:

- `go_server/internal/discovery/metrics.go` — bounded discovery-resolution and key-conflict Prometheus counters.
- `go_server/internal/discovery/repair/types.go` — audit, repair-plan, review, and comic-snapshot JSON contracts.
- `go_server/internal/discovery/repair/plan.go` — deterministic plan construction, signals, risk, validation, and logical fingerprinting.
- `go_server/internal/discovery/repair/plan_test.go` — planner, signals, review-gate, and fingerprint tests.
- `go_server/internal/repo/comics/discovery_audit.go` — query-only integrity audit and owner lookup.
- `go_server/internal/repo/comics/discovery_audit_test.go` — fixture audit and no-write tests.
- `go_server/internal/repo/comics/discovery_contract_test.go` — shared SQLite/PostgreSQL invariant suite.
- `go_server/cmd/dedupe/main_test.go` — CLI audit/export/validation tests.
- `go_server/cmd/dedupe/audit_live_test.go` — opt-in, read-only live-data audit.
- `go_server/cmd/db_maint/main_test.go` — reviewed apply, stale-plan, backup, and rollback tests.
- `docs/runbooks/discovery-data-repair.md` — exact rehearsal, production, and rollback runbook.

Files modified:

- `go_server/internal/identity/identity.go` — Unicode-only key fallback and scoped key-pair helper.
- `go_server/internal/identity/identity_test.go` — compatibility and CJK key tests.
- `go_server/domain/comic.go` — typed key-conflict and ambiguous-match errors.
- `go_server/internal/repo/comics/title_keys.go` — checked inserts, self exclusion, deleted-owner reclamation, and key audit support.
- `go_server/internal/repo/comics/sql_repository.go` — defensive write normalization and atomic create/update/delete wrappers.
- `go_server/internal/repo/comics/sql_repository_identity_test.go` — SQLite ownership and rollback tests.
- `go_server/internal/repo/comics/sql_repository_postgres_test.go` — temporary environment-gated PostgreSQL invocation of the shared contract; Workstream 3 replaces this file with its mandatory testcontainer contract runner.
- `go_server/internal/discovery/resolver.go` — explicit scoring, exclusion, ambiguity, and observation.
- `go_server/internal/discovery/resolver_test.go` — score matrix, margin, exclusion, and metric tests.
- `go_server/internal/usecase/comics.go` — transactional create/update checks and merge key transfer.
- `go_server/internal/usecase/comics_test.go` — normalized update, self exclusion, conflict, and merge tests.
- `go_server/api/route/comics.go` — stable HTTP 409 error mapping.
- `go_server/api/route/comics_contract_test.go` — public conflict payload contract.
- `go_server/internal/scrape/register.go` — resolver options and scrape conflict observation.
- `go_server/internal/scrape/scrape_test.go` — ambiguous scrape and normalized update tests.
- `go_server/cmd/import_comics_postgres/main.go` — normalize and preflight all rows, then replace comics and keys atomically.
- `go_server/cmd/import_comics_postgres/main_test.go` — normalized import, source collision, and deleted-key tests.
- `go_server/cmd/dedupe/main.go` — query-only audit/export/validate command; remove data mutation from this binary.
- `go_server/cmd/db_maint/main.go` — offline reviewed repair application with backup verification.
- `go_server/Makefile` — PostgreSQL discovery contract target and detailed help for the existing dedupe/maintenance commands.
- `Makefile` — root forwarding targets and deterministic backup destination support.

Files inspected but intentionally unchanged:

- `go_server/internal/scrape/audit_live_test.go` — supplies the opt-in live-test convention; publisher probing is unrelated to catalog repair.
- `go_server/internal/repo/sql/migrations/004_comic_title_keys.sql` — foreign-key parity is Workstream 3.

## Stable Interfaces Introduced by This Plan

```go
// internal/identity
type ScopedTitleKey struct {
	Scope    string `json:"scope"`
	TitleKey string `json:"title_key"`
}

func DiscoveryTitleKeyPairsForTitles(titles []string, comType int) []ScopedTitleKey
```

```go
// domain
var ErrDiscoveryKeyConflict = errors.New("discovery key conflict")
var ErrAmbiguousComicMatch = errors.New("ambiguous comic match")

type DiscoveryKeyConflictError struct {
	ComicID       int
	OwnerComicID  int
	Scope         string
	TitleKey      string
}

type AmbiguousComicMatchError struct {
	IncomingTitle string
	Scope         string
	MatchKind     string
	CandidateIDs  []int
	BestScore     int
	RunnerUpScore int
}
```

```go
// internal/discovery
type ResolveOptions struct {
	ExcludeComicID int
}

type Observer interface {
	ObserveResolution(outcome ResolutionOutcome, kind MatchKind)
	ObserveConflict(operation string)
}

type Resolver struct {
	observer Observer
}

func NewResolver(observer Observer) *Resolver
func DefaultObserver() Observer
func (r *Resolver) Resolve(
	ctx context.Context,
	repo domain.ComicRepository,
	incomingTitle string,
	comType int,
	options ResolveOptions,
) (domain.Comic, MatchKind, bool, error)
```

```go
// internal/repo/comics
func NewSQLiteComicRepositoryReadOnly(path string) (*SQLComicRepository, error)
func (r *SQLComicRepository) AuditDiscoveryIntegrity(
	ctx context.Context,
) (repair.Audit, error)
```

```go
// internal/discovery/repair
func BuildPlan(audit Audit, generatedAt time.Time) Plan
func ValidatePlan(plan Plan, current Audit) error
func FilterApproved(plan Plan, groupID string, allApproved bool) ([]Group, error)
```

### Task 1: Preserve Existing Keys and Add Unicode-Only Fallback

**Files:**
- Modify: `go_server/internal/identity/identity.go:17-20,117-120,240-257`
- Test: `go_server/internal/identity/identity_test.go`

**Interfaces:**
- Produces: `identity.ScopedTitleKey` and `identity.DiscoveryTitleKeyPairsForTitles`.
- Preserves: every non-empty key currently returned by `TitleMatchKey`.
- Consumed by: repository synchronization, import preflight, and repair planning.

- [ ] **Step 1: Write the failing compatibility and Unicode tests**

Add:

```go
func TestTitleMatchKeyUsesUnicodeOnlyWhenASCIIKeyIsEmpty(t *testing.T) {
	tests := map[string]string{
		"The executioner": "executioner",
		"Café terrace":    "cafterrace",
		"菜鳥白魔法師塞西爾本非我愛": "菜鳥白魔法師塞西爾本非我愛",
		"イジめてごっこ。":       "イジめてごっこ",
	}
	for title, want := range tests {
		if got := identity.TitleMatchKey(title); got != want {
			t.Fatalf("TitleMatchKey(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestDiscoveryTitleKeyPairsForTitlesPreservesScope(t *testing.T) {
	got := identity.DiscoveryTitleKeyPairsForTitles(
		[]string{"The executioner", "Executioner", "菜鳥白魔法師"},
		3,
	)
	want := []identity.ScopedTitleKey{
		{Scope: "series", TitleKey: "executioner"},
		{Scope: "series", TitleKey: "菜鳥白魔法師"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("scoped keys mismatch (-want +got):\n%s", diff)
	}
}
```

Add `github.com/google/go-cmp/cmp` to the test imports.

- [ ] **Step 2: Run the tests and confirm the new behavior is absent**

Run:

```bash
cd go_server
go test ./internal/identity -run 'TestTitleMatchKeyUsesUnicodeOnlyWhenASCIIKeyIsEmpty|TestDiscoveryTitleKeyPairsForTitlesPreservesScope' -count=1
```

Expected: FAIL because a CJK-only title still produces `""` and `ScopedTitleKey` is undefined.

- [ ] **Step 3: Add the fallback and scoped-pair helper**

Keep `nonMatchCharRe` unchanged and add:

```go
var nonUnicodeMatchCharRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

type ScopedTitleKey struct {
	Scope    string `json:"scope"`
	TitleKey string `json:"title_key"`
}
```

Replace `TitleMatchKey` with:

```go
func TitleMatchKey(title string) string {
	normalized := NormalizePrimaryTitle(title)
	normalized = leadingArticleRe.ReplaceAllString(normalized, "")
	asciiKey := nonMatchCharRe.ReplaceAllString(normalized, "")
	if asciiKey != "" {
		return asciiKey
	}
	return nonUnicodeMatchCharRe.ReplaceAllString(normalized, "")
}
```

Add:

```go
func DiscoveryTitleKeyPairsForTitles(titles []string, comType int) []ScopedTitleKey {
	seen := map[ScopedTitleKey]struct{}{}
	out := make([]ScopedTitleKey, 0, len(titles))
	for _, title := range titles {
		item := ScopedTitleKey{
			Scope:    DiscoveryScope(title, comType),
			TitleKey: DiscoveryTitleKey(title),
		}
		if item.TitleKey == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
```

Refactor `DiscoveryKeysForTitles` and `DiscoveryTitleKeysForTitles` to iterate this helper without changing their signatures.

- [ ] **Step 4: Run identity tests**

Run:

```bash
cd go_server
go test ./internal/identity -count=1
```

Expected: PASS. Existing ASCII, article, smart-quote, novel-scope, and sentence-case expectations remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add go_server/internal/identity/identity.go go_server/internal/identity/identity_test.go
git commit -m "feat: make discovery title keys unicode-safe"
```

### Task 2: Enforce Typed Key Ownership and Atomic Writes in SQLite

**Files:**
- Modify: `go_server/domain/comic.go:12-18`
- Modify: `go_server/internal/repo/comics/title_keys.go:14-28,198-228`
- Modify: `go_server/internal/repo/comics/sql_repository.go:241-398`
- Test: `go_server/internal/repo/comics/sql_repository_identity_test.go`

**Interfaces:**
- Produces: `domain.DiscoveryKeyConflictError`.
- Preserves: `errors.Is(err, domain.ErrDuplicateComic)` for exact duplicate callers.
- Guarantees: active create/update either persists the comic and every key or persists neither.

- [ ] **Step 1: Write the failing SQLite ownership contract**

Add:

```go
func TestSQLiteDiscoveryOwnershipIsTypedAndAtomic(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)

	owner, err := repo.Create(ctx, domain.Comic{
		Titles:  []string{"Alpha owner"},
		ComType: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := repo.Create(ctx, domain.Comic{
		Titles:  []string{"Beta candidate"},
		ComType: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	sameOwner := owner
	sameOwner.Titles = []string{"THE ALPHA OWNER"}
	if _, err := repo.Update(ctx, sameOwner); err != nil {
		t.Fatalf("self-owned key should update: %v", err)
	}

	conflicting := candidate
	conflicting.Titles = []string{"The alpha owner"}
	_, err = repo.Update(ctx, conflicting)
	var conflict *domain.DiscoveryKeyConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected typed conflict, got %v", err)
	}
	if conflict.ComicID != candidate.ID || conflict.OwnerComicID != owner.ID ||
		conflict.Scope != "series" || conflict.TitleKey != "alphaowner" {
		t.Fatalf("unexpected conflict: %#v", conflict)
	}

	stored, err := repo.Get(ctx, candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]string{"Beta candidate"}, stored.Titles); diff != "" {
		t.Fatalf("comic row changed despite conflict (-want +got):\n%s", diff)
	}
	beta, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{"betacandidate"})
	if err != nil || len(beta) != 1 || beta[0].ID != candidate.ID {
		t.Fatalf("candidate key did not roll back: %#v err=%v", beta, err)
	}
}

func TestSQLiteSoftDeleteReleasesAndUndeleteReacquiresKeys(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	owner, err := repo.Create(ctx, domain.Comic{Titles: []string{"Released key"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	owner.Deleted = true
	if _, err := repo.Update(ctx, owner); err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{"releasedkey"})
	if err != nil || len(found) != 0 {
		t.Fatalf("deleted owner retained a key: %#v err=%v", found, err)
	}

	replacement, err := repo.Create(ctx, domain.Comic{Titles: []string{"Released key"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	owner.Deleted = false
	if _, err := repo.Update(ctx, owner); !errors.Is(err, domain.ErrDiscoveryKeyConflict) {
		t.Fatalf("undelete should conflict with comic %d: %v", replacement.ID, err)
	}
	stored, err := repo.Get(ctx, owner.ID)
	if err != nil || !stored.Deleted {
		t.Fatalf("failed undelete should roll back: %#v err=%v", stored, err)
	}
}
```

Add `errors` and `github.com/google/go-cmp/cmp` to the test imports.

- [ ] **Step 2: Run the tests and verify the silent-conflict failure**

Run:

```bash
cd go_server
go test ./internal/repo/comics -run 'TestSQLiteDiscoveryOwnershipIsTypedAndAtomic|TestSQLiteSoftDeleteReleasesAndUndeleteReacquiresKeys' -count=1
```

Expected: FAIL because `ErrDiscoveryKeyConflict` is undefined and the current `INSERT OR IGNORE` permits a keyless update.

- [ ] **Step 3: Add the typed domain error**

Add to `domain/comic.go`:

```go
var ErrDiscoveryKeyConflict = errors.New("discovery key conflict")

type DiscoveryKeyConflictError struct {
	ComicID      int
	OwnerComicID int
	Scope        string
	TitleKey     string
}

func (e *DiscoveryKeyConflictError) Error() string {
	return fmt.Sprintf(
		"%s: %s:%s belongs to comic %d, cannot assign to comic %d",
		ErrDiscoveryKeyConflict,
		e.Scope,
		e.TitleKey,
		e.OwnerComicID,
		e.ComicID,
	)
}

func (e *DiscoveryKeyConflictError) Is(target error) bool {
	return target == ErrDiscoveryKeyConflict || target == ErrDuplicateComic
}
```

Add `fmt` to the imports. The custom `Is` method keeps exact-key create callers compatible while exposing the owner.

- [ ] **Step 4: Replace unchecked inserts with checked ownership**

Rename the insert constants and replace `upsertComicTitleKeySQL` with `insertComicTitleKeySQL` so no unchecked-upsert name remains:

```go
const (
	sqlInsertComicTitleKeySQLite = `INSERT OR IGNORE INTO comic_title_keys
		(comic_id, scope, title_key) VALUES (?, ?, ?)`
	sqlInsertComicTitleKeyPostgres = `INSERT INTO comic_title_keys
		(comic_id, scope, title_key) VALUES ($1, $2, $3)
		ON CONFLICT (scope, title_key) DO NOTHING`
)

func insertComicTitleKeySQL(dialect string) string {
	if dialect == dialectSQLite {
		return sqlInsertComicTitleKeySQLite
	}
	return sqlInsertComicTitleKeyPostgres
}
```

Delete `backfillComicTitleKeysIfEmpty`, `fastBackfillComicTitleKeys`, and the now-unused `isPostgresDB`; remove the resulting unused `log/slog` and `time` imports. Make `ensureComicTitleKeysSchema` schema-only:

```go
func ensureComicTitleKeysSchema(db *sql.DB) error {
	_, err := db.Exec(comicTitleKeysDDL)
	return err
}
```

Startup must never pick the first duplicate owner and silently leave the other comic keyless. Existing catalogs are audited and repaired offline; new writes use the checked path below.

Add dialect-aware owner lookup:

```go
type discoveryKeyOwner struct {
	ComicID int
	Deleted bool
}

func (r *SQLComicRepository) discoveryKeyOwner(
	ctx context.Context,
	scope string,
	titleKey string,
) (discoveryKeyOwner, bool, error) {
	deletedExpr := "c.deleted"
	query := `SELECT k.comic_id, ` + deletedExpr + `
		FROM comic_title_keys k
		INNER JOIN comics c ON c.id = k.comic_id
		WHERE k.scope = ` + r.placeholder(1) + `
		  AND k.title_key = ` + r.placeholder(2)
	var owner discoveryKeyOwner
	err := r.queryRow(ctx, query, scope, titleKey).Scan(&owner.ComicID, &owner.Deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return discoveryKeyOwner{}, false, nil
	}
	return owner, err == nil, err
}
```

Replace `insertDiscoveryKey` with:

```go
func (r *SQLComicRepository) insertDiscoveryKey(
	ctx context.Context,
	comicID int,
	scope string,
	titleKey string,
) error {
	result, err := r.exec(ctx, insertComicTitleKeySQL(r.dialect), comicID, scope, titleKey)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		return nil
	}
	owner, found, err := r.discoveryKeyOwner(ctx, scope, titleKey)
	if err != nil {
		return err
	}
	if found && owner.ComicID == comicID {
		return nil
	}
	if found && owner.Deleted {
		if _, err := r.exec(
			ctx,
			`DELETE FROM comic_title_keys WHERE scope = `+r.placeholder(1)+
				` AND title_key = `+r.placeholder(2)+
				` AND comic_id = `+r.placeholder(3),
			scope,
			titleKey,
			owner.ComicID,
		); err != nil {
			return err
		}
		return r.insertDiscoveryKey(ctx, comicID, scope, titleKey)
	}
	if found {
		return &domain.DiscoveryKeyConflictError{
			ComicID:      comicID,
			OwnerComicID: owner.ComicID,
			Scope:        scope,
			TitleKey:     titleKey,
		}
	}
	return fmt.Errorf("discovery key insert affected no rows for %s:%s", scope, titleKey)
}
```

Before deleting the current comic's keys, `SyncDiscoveryKeys` must preflight every `identity.DiscoveryTitleKeyPairsForTitles` pair. Ignore `owner.ComicID == comicID`, remove a deleted owner's stale key in the current transaction, and return `DiscoveryKeyConflictError` for another active owner. After preflight, delete current keys and call the checked insert for each pair.

- [ ] **Step 5: Normalize and transaction-wrap every repository write**

Add:

```go
func normalizeComicForWrite(comic domain.Comic) (domain.Comic, error) {
	comic.Titles = identity.NormalizeTitleVariants(comic.Titles, comic.ComType)
	if !comic.Deleted && len(identity.DiscoveryTitleKeyPairsForTitles(comic.Titles, comic.ComType)) == 0 {
		return domain.Comic{}, fmt.Errorf(
			"%w: active comic requires a title with a discovery key",
			domain.ErrInvalidComicPayload,
		)
	}
	return comic, nil
}
```

Split the existing SQL bodies into private `create`, `update`, and `delete` methods. Public methods start a transaction when `r.tx == nil`:

```go
func (r *SQLComicRepository) Update(
	ctx context.Context,
	comic domain.Comic,
) (domain.Comic, error) {
	if r.tx != nil {
		return r.update(ctx, comic)
	}
	var updated domain.Comic
	err := r.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		var err error
		updated, err = txRepo.Update(ctx, comic)
		return err
	})
	return updated, err
}

func (r *SQLComicRepository) Create(
	ctx context.Context,
	comic domain.Comic,
) (domain.Comic, error) {
	if r.tx != nil {
		return r.create(ctx, comic)
	}
	var created domain.Comic
	err := r.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		var err error
		created, err = txRepo.Create(ctx, comic)
		return err
	})
	return created, err
}

func (r *SQLComicRepository) Delete(ctx context.Context, id int) error {
	if r.tx != nil {
		return r.delete(ctx, id)
	}
	return r.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		return txRepo.Delete(ctx, id)
	})
}
```

In private create/update:

```go
comic, err := normalizeComicForWrite(comic)
if err != nil {
	return domain.Comic{}, err
}
```

After the row write:

```go
if comic.Deleted {
	if err := r.DeleteDiscoveryKeysForComic(ctx, comic.ID); err != nil {
		return domain.Comic{}, err
	}
} else if err := r.SyncDiscoveryKeys(ctx, comic.ID, comic.ComType, comic.Titles); err != nil {
	return domain.Comic{}, err
}
```

For SQLite create, set `comic.ID` from `LastInsertId` before this branch. For PostgreSQL create, set it from `RETURNING id`. Wrap standalone `SyncDiscoveryKeys` in `WithTx` using the same `r.tx` guard so maintenance callers cannot partially replace keys.

- [ ] **Step 6: Run repository and use-case regression tests**

Run:

```bash
cd go_server
go test ./internal/repo/comics ./internal/usecase ./internal/scrape -count=1
```

Expected: PASS. The two new tests prove conflict rollback and deleted-key release.

- [ ] **Step 7: Commit**

```bash
git add go_server/domain/comic.go \
  go_server/internal/repo/comics/title_keys.go \
  go_server/internal/repo/comics/sql_repository.go \
  go_server/internal/repo/comics/sql_repository_identity_test.go
git commit -m "feat: enforce discovery key ownership"
```

### Task 3: Run One Discovery Contract Against SQLite and PostgreSQL

This task supplies immediate parity coverage while Workstream 2 is landing.
Workstream 3 Task 5 supersedes the environment-skipped PostgreSQL wrapper and
`go-test-postgres-discovery` target with
`repository_contract_postgres_test.go` and the mandatory testcontainer matrix.
Do not preserve both PostgreSQL harnesses after Workstream 3 lands.

**Files:**
- Create: `go_server/internal/repo/comics/discovery_contract_test.go`
- Modify: `go_server/internal/repo/comics/sql_repository_postgres_test.go`
- Modify: `go_server/Makefile`

**Interfaces:**
- Consumes: typed conflicts and atomic repository writes from Task 2.
- Produces: `runDiscoveryContract(t, repo)` used by both dialect tests.
- Verifies: normalization, self ownership, conflict rollback, soft delete, hard delete, and CJK fallback.

- [ ] **Step 1: Extract a shared failing contract**

Create a helper with these subtests:

```go
func runDiscoveryContract(t *testing.T, repo *SQLComicRepository) {
	t.Helper()
	ctx := context.Background()
	prefix := fmt.Sprintf("contract-%d-", time.Now().UnixNano())

	t.Run("normalizes and stores every active key", func(t *testing.T) {
		created, err := repo.Create(ctx, domain.Comic{
			Titles:  []string{prefix + "ALPHA", prefix + "alpha alias"},
			ComType: 3,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, created.ID) })
		if created.Titles[0] != identity.NormalizeTitleForStorage(prefix+"ALPHA", 3, true) {
			t.Fatalf("title was not normalized: %#v", created.Titles)
		}
		for _, pair := range identity.DiscoveryTitleKeyPairsForTitles(created.Titles, created.ComType) {
			found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, pair.Scope, []string{pair.TitleKey})
			if err != nil || len(found) != 1 || found[0].ID != created.ID {
				t.Fatalf("missing key %#v: %#v err=%v", pair, found, err)
			}
		}
	})

	t.Run("conflict is typed and row plus old keys roll back", func(t *testing.T) {
		owner, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "owner"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "candidate"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = repo.Delete(ctx, owner.ID)
			_ = repo.Delete(ctx, candidate.ID)
		})
		candidate.Titles = owner.Titles
		_, err = repo.Update(ctx, candidate)
		var conflict *domain.DiscoveryKeyConflictError
		if !errors.As(err, &conflict) || conflict.OwnerComicID != owner.ID {
			t.Fatalf("expected owner %d conflict, got %#v / %v", owner.ID, conflict, err)
		}
		stored, err := repo.Get(ctx, candidate.ID)
		if err != nil || stored.Titles[0] != identity.NormalizeTitleForStorage(prefix+"candidate", 3, true) {
			t.Fatalf("rollback mismatch: %#v err=%v", stored, err)
		}
	})

	t.Run("self update is not a duplicate", func(t *testing.T) {
		comic, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "self"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, comic.ID) })
		comic.Titles = []string{strings.ToUpper(prefix + "self")}
		if _, err := repo.Update(ctx, comic); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("soft and hard delete release keys", func(t *testing.T) {
		comic, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "released"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		comic.Deleted = true
		if _, err := repo.Update(ctx, comic); err != nil {
			t.Fatal(err)
		}
		key := identity.DiscoveryTitleKey(prefix + "released")
		found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{key})
		if err != nil || len(found) != 0 {
			t.Fatalf("soft-deleted key remains: %#v err=%v", found, err)
		}
		if err := repo.Delete(ctx, comic.ID); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unicode-only title receives a key", func(t *testing.T) {
		comic, err := repo.Create(ctx, domain.Comic{Titles: []string{"菜鳥白魔法師" + prefix}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, comic.ID) })
		if identity.DiscoveryTitleKey(comic.Titles[0]) == "" {
			t.Fatal("unicode title produced an empty key")
		}
	})
}
```

- [ ] **Step 2: Invoke the contract for SQLite and PostgreSQL**

Add:

```go
func TestSQLiteDiscoveryContract(t *testing.T) {
	runDiscoveryContract(t, newTestSQLiteRepo(t))
}
```

Replace the single PostgreSQL lookup test body with:

```go
func TestPostgresDiscoveryContract(t *testing.T) {
	url := os.Getenv("COMICS_POSTGRES_URL")
	if url == "" {
		t.Skip("COMICS_POSTGRES_URL not set")
	}
	repo, err := NewPostgresComicRepository(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	runDiscoveryContract(t, repo)
}
```

- [ ] **Step 3: Run SQLite and the default PostgreSQL skip path**

Run:

```bash
cd go_server
go test ./internal/repo/comics -run 'TestSQLiteDiscoveryContract|TestPostgresDiscoveryContract' -count=1 -v
```

Expected: SQLite subtests PASS. PostgreSQL reports `SKIP: COMICS_POSTGRES_URL not set` when the variable is absent.

- [ ] **Step 4: Add and run the explicit PostgreSQL contract target**

Add to `go_server/Makefile`:

```make
## @quality:go-test-postgres-discovery Run discovery contract against COMICS_POSTGRES_URL
go-test-postgres-discovery:
	@test -n "$${COMICS_POSTGRES_URL}" || (echo "COMICS_POSTGRES_URL is required"; exit 2)
	go test ./internal/repo/comics -run TestPostgresDiscoveryContract -count=1 -v
```

Start an isolated database and run:

```bash
docker run --rm --name comics-discovery-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=comics_test \
  -p 55432:5432 \
  -d postgres:16-alpine
until docker exec comics-discovery-postgres pg_isready -U postgres -d comics_test; do sleep 1; done
cd go_server
COMICS_POSTGRES_URL='postgres://postgres:postgres@localhost:55432/comics_test?sslmode=disable' \
  make go-test-postgres-discovery
cd ..
docker stop comics-discovery-postgres
```

Expected: all five `TestPostgresDiscoveryContract` subtests PASS, then the container stops.

- [ ] **Step 5: Commit**

```bash
git add go_server/internal/repo/comics/discovery_contract_test.go \
  go_server/internal/repo/comics/sql_repository_postgres_test.go \
  go_server/Makefile
git commit -m "test: enforce discovery repository parity"
```

When rebasing this task after Workstream 3, keep
`discovery_contract_test.go`, register it with Workstream 3's shared
testcontainer harness, and skip edits to the deleted
`sql_repository_postgres_test.go` and temporary Make target.

### Task 4: Make Resolver Ambiguity Explicit and Observable

**Files:**
- Modify: `go_server/domain/comic.go`
- Create: `go_server/internal/discovery/metrics.go`
- Modify: `go_server/internal/discovery/resolver.go`
- Test: `go_server/internal/discovery/resolver_test.go`

**Interfaces:**
- Produces: `Resolver.Resolve` with `ResolveOptions.ExcludeComicID`.
- Produces metrics:
  - `comics_discovery_resolution_total{outcome,match_kind}`
  - `comics_discovery_key_conflicts_total{operation}`
- Allowed resolution outcomes: `matched`, `not_found`, `ambiguous`, `error`.
- Allowed match kinds: `none`, `discovery_key`, `title_prefix`, `title_tokens`.

- [ ] **Step 1: Replace the current winner-by-ID test with a score matrix**

Add tests for these exact cases:

```go
func TestResolverPolicy(t *testing.T) {
	tests := []struct {
		name      string
		incoming  string
		byKeys    []domain.Comic
		byToken   []domain.Comic
		excludeID int
		wantID    int
		wantKind  discovery.MatchKind
		ambiguous bool
	}{
		{
			name:     "exact stored text beats article variant",
			incoming: "Executioner",
			byKeys: []domain.Comic{
				{ID: 1071, Titles: []string{"The executioner"}, ComType: 3},
				{ID: 34458, Titles: []string{"Executioner"}, ComType: 3},
			},
			wantID:   34458,
			wantKind: discovery.MatchDiscoveryKey,
		},
		{
			name:     "equal exact relations are ambiguous",
			incoming: "Executioner",
			byKeys: []domain.Comic{
				{ID: 20, Titles: []string{"The executioner"}, ComType: 3},
				{ID: 10, Titles: []string{"The executioner"}, ComType: 3, Track: true, CurrentChap: 999},
			},
			ambiguous: true,
		},
		{
			name:      "update excludes itself",
			incoming:  "Executioner",
			byKeys:    []domain.Comic{{ID: 10, Titles: []string{"Executioner"}, ComType: 3}},
			excludeID: 10,
		},
		{
			name:     "prefix beats no relation",
			incoming: "The strongest assassin gets transferr",
			byToken: []domain.Comic{
				{ID: 9, Titles: []string{"The strongest assassin gets transferred to another world"}, ComType: 3},
			},
			wantID:   9,
			wantKind: discovery.MatchTitlePrefix,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolver := discovery.NewResolver(discovery.NoopObserver{})
			repo := &stubRepo{byKeys: tc.byKeys, byToken: tc.byToken}
			comic, kind, found, err := resolver.Resolve(
				context.Background(),
				repo,
				tc.incoming,
				3,
				discovery.ResolveOptions{ExcludeComicID: tc.excludeID},
			)
			if tc.ambiguous {
				var ambiguous *domain.AmbiguousComicMatchError
				if !errors.As(err, &ambiguous) {
					t.Fatalf("expected ambiguity, got %v", err)
				}
				return
			}
			if err != nil || comic.ID != tc.wantID || kind != tc.wantKind || found != (tc.wantID != 0) {
				t.Fatalf("got id=%d kind=%s found=%v err=%v", comic.ID, kind, found, err)
			}
		})
	}
}
```

Add a metric test using a fresh registry:

```go
func TestResolverRecordsBoundedAmbiguityMetric(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := discovery.NewPrometheusObserver(registry)
	resolver := discovery.NewResolver(observer)
	repo := &stubRepo{byKeys: []domain.Comic{
		{ID: 1, Titles: []string{"The executioner"}, ComType: 3},
		{ID: 2, Titles: []string{"The executioner"}, ComType: 3},
	}}
	_, _, _, _ = resolver.Resolve(
		context.Background(),
		repo,
		"Executioner",
		3,
		discovery.ResolveOptions{},
	)
	err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(`# HELP comics_discovery_resolution_total Discovery resolution outcomes.
# TYPE comics_discovery_resolution_total counter
comics_discovery_resolution_total{match_kind="discovery_key",outcome="ambiguous"} 1
`),
		"comics_discovery_resolution_total",
	)
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the resolver tests and verify current tie-breaking fails**

Run:

```bash
cd go_server
go test ./internal/discovery -count=1
```

Expected: FAIL because `Resolver`, `ResolveOptions`, `AmbiguousComicMatchError`, and the metrics observer do not exist.

- [ ] **Step 3: Add the typed ambiguity error**

Add to `domain/comic.go`:

```go
var ErrAmbiguousComicMatch = errors.New("ambiguous comic match")

type AmbiguousComicMatchError struct {
	IncomingTitle string
	Scope         string
	MatchKind     string
	CandidateIDs  []int
	BestScore     int
	RunnerUpScore int
}

func (e *AmbiguousComicMatchError) Error() string {
	return fmt.Sprintf(
		"%s: %q in %s scope matched comics %v with scores %d and %d",
		ErrAmbiguousComicMatch,
		e.IncomingTitle,
		e.Scope,
		e.CandidateIDs,
		e.BestScore,
		e.RunnerUpScore,
	)
}

func (e *AmbiguousComicMatchError) Is(target error) bool {
	return target == ErrAmbiguousComicMatch
}
```

- [ ] **Step 4: Add injectable bounded metrics**

In `metrics.go`, define:

```go
type ResolutionOutcome string

const (
	OutcomeMatched   ResolutionOutcome = "matched"
	OutcomeNotFound  ResolutionOutcome = "not_found"
	OutcomeAmbiguous ResolutionOutcome = "ambiguous"
	OutcomeError     ResolutionOutcome = "error"
)

type Observer interface {
	ObserveResolution(outcome ResolutionOutcome, kind MatchKind)
	ObserveConflict(operation string)
}

type NoopObserver struct{}

func (NoopObserver) ObserveResolution(ResolutionOutcome, MatchKind) {}
func (NoopObserver) ObserveConflict(string)                         {}

type PrometheusObserver struct {
	resolutionTotal *prometheus.CounterVec
	conflictTotal   *prometheus.CounterVec
}

func NewPrometheusObserver(registerer prometheus.Registerer) *PrometheusObserver {
	observer := &PrometheusObserver{
		resolutionTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "resolution_total",
			Help:      "Discovery resolution outcomes.",
		}, []string{"outcome", "match_kind"}),
		conflictTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "comics",
			Subsystem: "discovery",
			Name:      "key_conflicts_total",
			Help:      "Typed discovery key conflicts by catalog operation.",
		}, []string{"operation"}),
	}
	registerer.MustRegister(observer.resolutionTotal, observer.conflictTotal)
	return observer
}

func (o *PrometheusObserver) ObserveResolution(outcome ResolutionOutcome, kind MatchKind) {
	if kind == MatchNone {
		kind = MatchKind("none")
	}
	o.resolutionTotal.WithLabelValues(string(outcome), string(kind)).Inc()
}

func (o *PrometheusObserver) ObserveConflict(operation string) {
	o.conflictTotal.WithLabelValues(operation).Inc()
}

var defaultObserver Observer = NewPrometheusObserver(prometheus.DefaultRegisterer)

func DefaultObserver() Observer {
	return defaultObserver
}
```

Only fixed enum values and fixed operation names enter labels; comic IDs, titles, publishers, and URLs never become labels.

- [ ] **Step 5: Implement identity-only ranking and ambiguity margin**

Use these constants:

```go
const (
	scoreExactText        = 15
	scoreExactKey         = 10
	scoreTitleTokens      = 8
	scoreTitlePrefix      = 6
	minimumAcceptedScore  = 6
	minimumWinningMargin  = 2
)
```

For each comic, use the highest relationship score across its titles, not a sum. Remove the current `Track`, `CurrentChap`, and lower-ID winner bonuses. Sort scored candidates by score descending and ID ascending only to make error output deterministic.

The decision function is:

```go
func chooseCandidate(
	incoming string,
	scope string,
	kind MatchKind,
	candidates []domain.Comic,
	comType int,
) (domain.Comic, bool, error) {
	scored := scoreCandidates(candidates, incoming, comType)
	if len(scored) == 0 || scored[0].Score < minimumAcceptedScore {
		return domain.Comic{}, false, nil
	}
	if len(scored) > 1 && scored[0].Score-scored[1].Score < minimumWinningMargin {
		ids := []int{scored[0].Comic.ID, scored[1].Comic.ID}
		for _, item := range scored[2:] {
			if scored[0].Score-item.Score >= minimumWinningMargin {
				break
			}
			ids = append(ids, item.Comic.ID)
		}
		return domain.Comic{}, false, &domain.AmbiguousComicMatchError{
			IncomingTitle: incoming,
			Scope:         scope,
			MatchKind:     string(kind),
			CandidateIDs:  ids,
			BestScore:     scored[0].Score,
			RunnerUpScore: scored[1].Score,
		}
	}
	return scored[0].Comic, true, nil
}
```

Filter `options.ExcludeComicID` from both exact-key and token candidate lists before scoring. `Resolver.Resolve` records exactly one metric on every return path.

- [ ] **Step 6: Run resolver and full Go tests**

Run:

```bash
cd go_server
go test ./internal/discovery -count=1
go test ./... -count=1
```

Expected: PASS. The ambiguity test reports candidate IDs in deterministic ascending-ID order when scores tie.

- [ ] **Step 7: Commit**

```bash
git add go_server/domain/comic.go \
  go_server/internal/discovery/metrics.go \
  go_server/internal/discovery/resolver.go \
  go_server/internal/discovery/resolver_test.go
git commit -m "feat: reject ambiguous discovery matches"
```

### Task 5: Normalize and De-duplicate Create, Update, and Merge

**Files:**
- Modify: `go_server/internal/usecase/comics.go:20-30,63-102,145-180,299-337`
- Modify: `go_server/internal/usecase/comics_test.go`
- Modify: `go_server/api/route/comics.go:320-338`
- Modify: `go_server/api/route/comics_contract_test.go`

**Interfaces:**
- Consumes: `Resolver.Resolve`, typed conflict errors, and atomic repository writes.
- Produces: HTTP 409 payloads with codes `discovery_key_conflict` and `ambiguous_comic_match`.
- Guarantees: update excludes its own ID; merge releases the duplicate's keys before assigning them to the canonical row inside one transaction.

- [ ] **Step 1: Write failing use-case tests**

Add:

```go
func TestComicServiceUpdateNormalizesAndExcludesSelf(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryComicRepo()
	svc := NewComicService(repo, nil)
	created, err := svc.Create(ctx, domain.Comic{Titles: []string{"SELF TITLE"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	titles := []string{"self title", "SELF ALIAS"}
	updated, err := svc.Update(ctx, created.ID, domain.ComicPatch{Titles: &titles})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]string{"Self title", "Self alias"}, updated.Titles); diff != "" {
		t.Fatalf("normalized titles mismatch (-want +got):\n%s", diff)
	}
}

func TestComicServiceUpdateRejectsAnotherOwner(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryComicRepo()
	svc := NewComicService(repo, nil)
	owner, err := svc.Create(ctx, domain.Comic{Titles: []string{"Owned title"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := svc.Create(ctx, domain.Comic{Titles: []string{"Candidate title"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	titles := []string{"The owned title"}
	_, err = svc.Update(ctx, candidate.ID, domain.ComicPatch{Titles: &titles})
	var conflict *domain.DiscoveryKeyConflictError
	if !errors.As(err, &conflict) || conflict.OwnerComicID != owner.ID {
		t.Fatalf("expected owner %d conflict, got %#v / %v", owner.ID, conflict, err)
	}
}

func TestComicServiceMergeNormalizesAndTransfersKeys(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryComicRepo()
	svc := NewComicService(repo, nil)
	base, err := svc.Create(ctx, domain.Comic{Titles: []string{"BASE TITLE"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := svc.Create(ctx, domain.Comic{Titles: []string{"DUPLICATE ALIAS"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	merged, err := svc.Merge(ctx, base.ID, duplicate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]string{"Base title", "Duplicate alias"}, merged.Titles); diff != "" {
		t.Fatalf("merged titles mismatch (-want +got):\n%s", diff)
	}
}
```

Update `memoryComicRepo` to honor `ResolveOptions` only through the real resolver; no new repository method is required.

- [ ] **Step 2: Write failing HTTP conflict contracts**

Add route tests that create two comics, update the second to the first title, and assert:

```go
if response.Code != http.StatusConflict {
	t.Fatalf("expected 409, got %d: %s", response.Code, response.Body.String())
}
var body struct {
	Code         string `json:"code"`
	OwnerComicID int    `json:"owner_comic_id"`
	Scope        string `json:"scope"`
	TitleKey     string `json:"title_key"`
}
if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
	t.Fatal(err)
}
if body.Code != "discovery_key_conflict" || body.OwnerComicID != owner.ID {
	t.Fatalf("unexpected conflict body: %#v", body)
}
```

Update `TestComicsContractListSearchAndDuplicate` to expect HTTP 409 and `code == "discovery_key_conflict"` for an exact-key duplicate. Fuzzy duplicate matches remain `ErrDuplicateComic` and retain their existing bad-request mapping.

Add a direct handler test with `AmbiguousComicMatchError` and assert code `ambiguous_comic_match`, HTTP 409, and sorted `candidate_ids`.

- [ ] **Step 3: Run focused tests**

Run:

```bash
cd go_server
go test ./internal/usecase ./api/route -run 'UpdateNormalizes|UpdateRejectsAnotherOwner|MergeNormalizes|DiscoveryConflict|Ambiguous' -count=1
```

Expected: FAIL because update does not normalize or resolve duplicates and the route maps conflicts to 500.

- [ ] **Step 4: Inject one resolver and observer into the service**

Change the service fields and constructors to:

```go
type ComicService struct {
	repo        domain.ComicRepository
	scrapeCoord *scrape.Coordinator
	resolver    *discovery.Resolver
	observer    discovery.Observer
}

func NewComicService(repo domain.ComicRepository, scrapeCoord *scrape.Coordinator) *ComicService {
	return NewComicServiceWithObserver(repo, scrapeCoord, discovery.DefaultObserver())
}

func NewComicServiceWithObserver(
	repo domain.ComicRepository,
	scrapeCoord *scrape.Coordinator,
	observer discovery.Observer,
) *ComicService {
	if scrapeCoord == nil {
		scrapeCoord = scrape.NewCoordinator(repo)
	}
	if observer == nil {
		observer = discovery.NoopObserver{}
	}
	return &ComicService{
		repo:        repo,
		scrapeCoord: scrapeCoord,
		resolver:    discovery.NewResolver(observer),
		observer:    observer,
	}
}
```

- [ ] **Step 5: Make create and update resolve inside the write transaction**

Add:

```go
func (s *ComicService) rejectDuplicate(
	ctx context.Context,
	repo domain.ComicRepository,
	comic domain.Comic,
	excludeID int,
) error {
	for _, title := range comic.Titles {
		existing, kind, found, err := s.resolver.Resolve(
			ctx,
			repo,
			title,
			comic.ComType,
			discovery.ResolveOptions{ExcludeComicID: excludeID},
		)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if kind == discovery.MatchDiscoveryKey {
			return &domain.DiscoveryKeyConflictError{
				ComicID:      excludeID,
				OwnerComicID: existing.ID,
				Scope:        identity.DiscoveryScope(title, comic.ComType),
				TitleKey:     identity.DiscoveryTitleKey(title),
			}
		}
		return domain.ErrDuplicateComic
	}
	return nil
}
```

Create normalizes first, then calls `rejectDuplicate` with `excludeID=0` and
calls `repo.Create`, both inside `s.repo.WithTx`; a create conflict therefore
serializes `comic_id: 0` because no row has been allocated. Update gets the
current row, applies the patch, normalizes using the final `ComType`, skips
duplicate resolution only when the final row is deleted, calls
`rejectDuplicate` with `excludeID=id`, and calls `repo.Update`, all inside one
transaction.

When create/update returns `ErrDiscoveryKeyConflict`, call:

```go
s.observer.ObserveConflict("create")
```

or:

```go
s.observer.ObserveConflict("update")
```

Use only the fixed operation values `create`, `update`, and `merge`.

- [ ] **Step 6: Transfer duplicate keys safely during merge**

Inside the existing merge transaction, perform this order:

```go
merged = mergeComicValues(base, duplicate)
merged.Titles = identity.NormalizeTitleVariants(merged.Titles, merged.ComType)
if len(merged.Titles) == 0 {
	return fmt.Errorf("%w: merged comic has no valid title", domain.ErrInvalidComicPayload)
}
if err := repo.DeleteDiscoveryKeysForComic(ctx, mergingID); err != nil {
	return err
}
if _, err := repo.Update(ctx, merged); err != nil {
	return err
}
return repo.Delete(ctx, mergingID)
```

If a third comic owns any merged key, the typed conflict rolls the entire merge back, including the duplicate-key deletion. Record `merge` conflict metrics on return.

- [ ] **Step 7: Map typed errors to stable 409 responses**

Before generic duplicate and internal-error handling in `writeComicWithStatus`, add:

```go
var conflict *domain.DiscoveryKeyConflictError
if errors.As(err, &conflict) {
	c.JSON(http.StatusConflict, gin.H{
		"code":            "discovery_key_conflict",
		"message":         "Discovery key is owned by another comic",
		"owner_comic_id":  conflict.OwnerComicID,
		"scope":           conflict.Scope,
		"title_key":       conflict.TitleKey,
	})
	return
}
var ambiguous *domain.AmbiguousComicMatchError
if errors.As(err, &ambiguous) {
	c.JSON(http.StatusConflict, gin.H{
		"code":          "ambiguous_comic_match",
		"message":       "Title matches multiple comics",
		"candidate_ids": ambiguous.CandidateIDs,
		"match_kind":    ambiguous.MatchKind,
	})
	return
}
```

Workstream 1 makes these mutation routes admin-only, satisfying the approved rule that owner IDs are exposed only to authorized admins.

- [ ] **Step 8: Run focused and full tests**

Run:

```bash
cd go_server
go test ./internal/usecase ./api/route -count=1
go test ./... -count=1
```

Expected: PASS. Existing chapter timestamp, cover precedence, merge value, and HTTP contract tests remain green.

- [ ] **Step 9: Commit**

```bash
git add go_server/internal/usecase/comics.go \
  go_server/internal/usecase/comics_test.go \
  go_server/api/route/comics.go \
  go_server/api/route/comics_contract_test.go
git commit -m "fix: normalize catalog mutations safely"
```

### Task 6: Normalize Scrape and PostgreSQL Import Writes

**Files:**
- Modify: `go_server/internal/scrape/register.go:23-69`
- Modify: `go_server/internal/scrape/scrape_test.go`
- Modify: `go_server/cmd/import_comics_postgres/main.go:18-45,113-246`
- Modify: `go_server/cmd/import_comics_postgres/main_test.go`

**Interfaces:**
- Consumes: resolver options, identity scoped keys, and typed conflicts.
- Produces: `prepareImportComics([]sourceComic) ([]preparedComic, error)`.
- Guarantees: Postgres replacement inserts normalized comics and keys in one transaction, or leaves the target untouched.

- [ ] **Step 1: Add failing scraper ambiguity and normalization tests**

Add a repository stub returning two equal resolver candidates and assert:

```go
err := registrar.Register(context.Background(), ScrapedComic{
	Title:   "The executioner",
	Chapter: "10",
	ComType: "manhwa",
}, PublisherAsura)
if !errors.Is(err, domain.ErrAmbiguousComicMatch) {
	t.Fatalf("expected ambiguous scrape to stop, got %v", err)
}
if stub.created != 0 || stub.updated != 0 {
	t.Fatalf("ambiguous scrape wrote data: %#v", stub)
}
```

Extend the existing scrape repository test so an uppercase incoming alias is stored as sentence case after update.

- [ ] **Step 2: Add failing import preparation tests**

Add:

```go
func TestPrepareImportComicsNormalizesAndSkipsDeletedKeys(t *testing.T) {
	prepared, err := prepareImportComics([]sourceComic{
		{ID: 1, Titles: "THE HERO|The hero alias", ComType: 3},
		{ID: 2, Titles: "", ComType: 3, Deleted: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared[0].Titles != "The hero|The hero alias" || len(prepared[0].Keys) != 2 {
		t.Fatalf("unexpected active import: %#v", prepared[0])
	}
	if len(prepared[1].Keys) != 0 {
		t.Fatalf("deleted comic received keys: %#v", prepared[1].Keys)
	}
}

func TestPrepareImportComicsRejectsSourceKeyConflict(t *testing.T) {
	_, err := prepareImportComics([]sourceComic{
		{ID: 10, Titles: "The owner", ComType: 3},
		{ID: 11, Titles: "Owner", ComType: 3},
	})
	var conflict *domain.DiscoveryKeyConflictError
	if !errors.As(err, &conflict) || conflict.OwnerComicID != 10 || conflict.ComicID != 11 {
		t.Fatalf("expected 10 -> 11 conflict, got %#v / %v", conflict, err)
	}
}
```

- [ ] **Step 3: Run the focused tests**

Run:

```bash
cd go_server
go test ./internal/scrape ./cmd/import_comics_postgres -run 'Ambiguous|PrepareImport|Normalize' -count=1
```

Expected: FAIL because scraper registration uses the old resolver function and import preparation does not exist.

- [ ] **Step 4: Use the explicit resolver from scraper registration**

Add a resolver and observer to `Registrar`, initialized with `discovery.DefaultObserver()`. Resolve with an empty exclusion:

```go
existing, _, found, err := r.resolver.Resolve(
	ctx,
	txRepo,
	normalized.Titles[0],
	normalized.ComType,
	discovery.ResolveOptions{},
)
```

Return ambiguity and key-conflict errors without creating or updating. When create/update returns `ErrDiscoveryKeyConflict`, call `r.observer.ObserveConflict("scrape")`. Keep `normalizeScrapedComic` and `mergeTitleVariants` as the first normalization layer; repository normalization remains the defensive final layer.

- [ ] **Step 5: Preflight and normalize the complete import in memory**

Add:

```go
type preparedComic struct {
	sourceComic
	Titles string
	Keys   []identity.ScopedTitleKey
}

func prepareImportComics(source []sourceComic) ([]preparedComic, error) {
	owners := map[identity.ScopedTitleKey]int{}
	prepared := make([]preparedComic, 0, len(source))
	for _, comic := range source {
		titles := identity.NormalizeTitleVariants(splitImportTitles(comic.Titles), comic.ComType)
		keys := identity.DiscoveryTitleKeyPairsForTitles(titles, comic.ComType)
		if !comic.Deleted && len(keys) == 0 {
			return nil, fmt.Errorf(
				"%w: active source comic %d has no discovery key",
				domain.ErrInvalidComicPayload,
				comic.ID,
			)
		}
		if comic.Deleted {
			keys = nil
		}
		for _, key := range keys {
			if ownerID, exists := owners[key]; exists && ownerID != comic.ID {
				return nil, &domain.DiscoveryKeyConflictError{
					ComicID:      comic.ID,
					OwnerComicID: ownerID,
					Scope:        key.Scope,
					TitleKey:     key.TitleKey,
				}
			}
			owners[key] = comic.ID
		}
		prepared = append(prepared, preparedComic{
			sourceComic: comic,
			Titles:      strings.Join(titles, "|"),
			Keys:        keys,
		})
	}
	return prepared, nil
}
```

Call this before starting any target transaction.

- [ ] **Step 6: Replace comics and keys in one PostgreSQL transaction**

Within `replaceTargetComics`:

1. Create timestamped backup tables for both `comics` and `comic_title_keys` when target rows exist.
2. Run `TRUNCATE TABLE comic_title_keys, comics RESTART IDENTITY`.
3. Insert each normalized comic.
4. Insert each prepared key with plain `INSERT`; in-memory preflight guarantees uniqueness.
5. Set `comic_id_seq`.
6. Verify active comics have key owners and deleted comics have none before commit.
7. Commit only after all checks pass.

Delete the post-commit repository reopen and `SyncDiscoveryKeys` loop. Extend `verifyCounts`:

```go
type verifyCounts struct {
	Total              int
	Active             int
	Tracked            int
	CoverInvisible     int
	DiscoveryOwners    int
	KeylessActive      int
	DeletedKeyRows     int
}
```

Require `KeylessActive == 0`, `DeletedKeyRows == 0`, and `DiscoveryOwners == Active` for the repaired source. A pre-repair import now fails loudly on comic 4561 or the first conflict instead of creating a partial key table.

- [ ] **Step 7: Run scraper, importer, and repository tests**

Run:

```bash
cd go_server
go test ./internal/scrape ./cmd/import_comics_postgres ./internal/repo/comics -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add go_server/internal/scrape/register.go \
  go_server/internal/scrape/scrape_test.go \
  go_server/cmd/import_comics_postgres/main.go \
  go_server/cmd/import_comics_postgres/main_test.go
git commit -m "fix: normalize scraper and postgres imports"
```

### Task 7: Add Query-Only Audit for All 346 Records

**Files:**
- Create: `go_server/internal/discovery/repair/types.go`
- Create: `go_server/internal/repo/comics/discovery_audit.go`
- Create: `go_server/internal/repo/comics/discovery_audit_test.go`
- Modify: `go_server/cmd/dedupe/main.go`
- Create: `go_server/cmd/dedupe/main_test.go`
- Create: `go_server/cmd/dedupe/audit_live_test.go`

**Interfaces:**
- Produces: `repair.Audit`, including all active keyless records and owner snapshots.
- Produces: `NewSQLiteComicRepositoryReadOnly`, which never calls schema, index, or backfill setup.
- Audit classifications: `active_conflict`, `restorable`, `blocked_no_key`.

- [ ] **Step 1: Define the audit JSON contract**

Create:

```go
package repair

import "comics/internal/identity"

type ComicSnapshot struct {
	ID          int      `json:"id"`
	Titles      []string `json:"titles"`
	CurrentChap int      `json:"current_chap"`
	ComType     int      `json:"com_type"`
	Track       bool     `json:"track"`
	Deleted     bool     `json:"deleted"`
}

type KeyOwner struct {
	Key     identity.ScopedTitleKey `json:"key"`
	Comic  ComicSnapshot           `json:"comic"`
}

type KeylessRecord struct {
	Comic        ComicSnapshot             `json:"comic"`
	ProposedKeys []identity.ScopedTitleKey `json:"proposed_keys"`
	Owners       []KeyOwner                `json:"owners"`
	Classification string                  `json:"classification"`
}

type AuditSummary struct {
	TotalComics     int `json:"total_comics"`
	ActiveComics    int `json:"active_comics"`
	KeyRows         int `json:"key_rows"`
	ActiveKeyOwners int `json:"active_key_owners"`
	KeylessActive   int `json:"keyless_active"`
	ActiveConflicts int `json:"active_conflicts"`
	Restorable      int `json:"restorable"`
	BlockedNoKey    int `json:"blocked_no_key"`
	DeletedKeyRows  int `json:"deleted_key_rows"`
	OrphanKeyRows   int `json:"orphan_key_rows"`
}

type Audit struct {
	DatabaseFingerprint string          `json:"database_fingerprint"`
	Summary             AuditSummary    `json:"summary"`
	Records             []KeylessRecord `json:"records"`
}
```

The fingerprint is SHA-256 over canonical, ordered comic fields and ordered title-key rows, not raw SQLite bytes. It therefore includes committed WAL-visible data and is stable across `.backup` copies.

- [ ] **Step 2: Write fixture tests that prove audit is read-only**

Create a SQLite fixture with one owner, one conflicting keyless comic inserted by raw SQL, one CJK keyless comic, one blank-title comic, one deleted stale key, and one orphan key. Before and after `AuditDiscoveryIntegrity`, compare:

```go
func sqliteSchemaAndDataHash(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`
		SELECT type, name, tbl_name, sql FROM sqlite_master
		UNION ALL
		SELECT 'comic', CAST(id AS TEXT), titles, CAST(deleted AS TEXT) FROM comics
		UNION ALL
		SELECT 'key', CAST(comic_id AS TEXT), scope, title_key FROM comic_title_keys
		ORDER BY 1, 2, 3, 4`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	hash := sha256.New()
	for rows.Next() {
		var a, b, c string
		var d sql.NullString
		if err := rows.Scan(&a, &b, &c, &d); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(hash, "%q\x00%q\x00%q\x00%q\n", a, b, c, d.String)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
```

Assert the before/after hash is identical and summary values match the fixture exactly.

- [ ] **Step 3: Run the audit tests and verify the API is absent**

Run:

```bash
cd go_server
go test ./internal/repo/comics -run DiscoveryAudit -count=1
```

Expected: FAIL because the read-only constructor and audit method do not exist.

- [ ] **Step 4: Add a constructor that cannot initialize or backfill**

Implement:

```go
func NewSQLiteComicRepositoryReadOnly(path string) (*SQLComicRepository, error) {
	if strings.TrimSpace(path) == "" {
		path = findSQLiteComicDB()
	}
	if path == "" {
		return nil, fmt.Errorf("comic sqlite database not found")
	}
	dsn := "file:" + filepath.ToSlash(path) + "?mode=ro&_pragma=query_only%3d1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLComicRepository{
		db:      db,
		dialect: dialectSQLite,
		ownsDB:  true,
	}, nil
}
```

Do not call `configureSQLiteDB`, `ensureSQLitePerformanceIndexes`, or `ensureComicTitleKeysSchema`.

- [ ] **Step 5: Implement deterministic audit queries**

`AuditDiscoveryIntegrity` must:

1. Run `PRAGMA integrity_check` and fail unless the result is `ok`.
2. Count totals, active rows, key rows, active key owners, deleted key rows, and orphans.
3. Select every active comic with no key row ordered by ID.
4. Compute proposed scoped keys with Task 1 identity helpers.
5. Load each proposed key's owner and complete comic snapshot.
6. Classify records:
   - no proposed keys: `blocked_no_key`
   - at least one active other owner: `active_conflict`
   - no active other owner: `restorable`
7. Compute the logical SHA-256 fingerprint from ordered rows.
8. Return records and owners in deterministic ID/key order.

The method performs only `PRAGMA integrity_check` and `SELECT`.

- [ ] **Step 6: Turn dedupe audit into a query-only CLI**

Support:

```text
--db path
--audit
```

For `--audit`, use `NewSQLiteComicRepositoryReadOnly`. Remove the current `--apply` path from `cmd/dedupe`; data mutation belongs only to offline `db_maint`. Task 8 adds plan export and validation after the versioned planner exists.

Audit output format:

```go
fmt.Printf(
	"discovery audit: total=%d active=%d keys=%d active_owners=%d keyless=%d conflicts=%d restorable=%d blocked=%d deleted_keys=%d orphan_keys=%d\n",
	s.TotalComics,
	s.ActiveComics,
	s.KeyRows,
	s.ActiveKeyOwners,
	s.KeylessActive,
	s.ActiveConflicts,
	s.Restorable,
	s.BlockedNoKey,
	s.DeletedKeyRows,
	s.OrphanKeyRows,
)
```

- [ ] **Step 7: Add the opt-in live audit test**

Follow the existing scrape live-test convention:

```go
func TestAuditLiveDiscoveryIntegrity(t *testing.T) {
	if os.Getenv("AUDIT_DISCOVERY") != "1" {
		t.Skip("set AUDIT_DISCOVERY=1 to inspect the live catalog read-only")
	}
	path := os.Getenv("COMICS_SQLITE_PATH")
	if path == "" {
		path = "../../../db/comics.db"
	}
	before := fileDigestSet(t, path)
	repo, err := comics.NewSQLiteComicRepositoryReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := repo.AuditDiscoveryIntegrity(context.Background())
	closeErr := repo.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	after := fileDigestSet(t, path)
	if diff := cmp.Diff(before, after); diff != "" {
		t.Fatalf("audit changed SQLite files (-before +after):\n%s", diff)
	}
	t.Logf(
		"total=%d active=%d keys=%d keyless=%d conflicts=%d restorable=%d blocked=%d deleted_keys=%d orphan_keys=%d",
		audit.Summary.TotalComics,
		audit.Summary.ActiveComics,
		audit.Summary.KeyRows,
		audit.Summary.KeylessActive,
		audit.Summary.ActiveConflicts,
		audit.Summary.Restorable,
		audit.Summary.BlockedNoKey,
		audit.Summary.DeletedKeyRows,
		audit.Summary.OrphanKeyRows,
	)
}
```

`fileDigestSet` hashes the main database plus existing `-wal` and `-shm` files and records file mode, size, and modification time.

- [ ] **Step 8: Run fixture and live audits**

Run:

```bash
cd go_server
go test ./internal/repo/comics ./cmd/dedupe -run DiscoveryAudit -count=1
AUDIT_DISCOVERY=1 COMICS_SQLITE_PATH=../db/comics.db \
  go test ./cmd/dedupe -run TestAuditLiveDiscoveryIntegrity -count=1 -v
```

Expected live log:

```text
total=31309 active=31308 keys=31892 keyless=346 conflicts=341 restorable=4 blocked=1 deleted_keys=0 orphan_keys=0
```

The database, WAL, and SHM digest set is unchanged.

- [ ] **Step 9: Commit**

```bash
git add go_server/internal/discovery/repair/types.go \
  go_server/internal/repo/comics/discovery_audit.go \
  go_server/internal/repo/comics/discovery_audit_test.go \
  go_server/cmd/dedupe/main.go \
  go_server/cmd/dedupe/main_test.go \
  go_server/cmd/dedupe/audit_live_test.go
git commit -m "feat: add read-only discovery audit"
```

### Task 8: Build a Reviewed Repair Plan and Guarded Offline Apply

**Files:**
- Create: `go_server/internal/discovery/repair/plan.go`
- Create: `go_server/internal/discovery/repair/plan_test.go`
- Modify: `go_server/cmd/dedupe/main.go`
- Modify: `go_server/cmd/dedupe/main_test.go`
- Modify: `go_server/cmd/db_maint/main.go`
- Create: `go_server/cmd/db_maint/main_test.go`

**Interfaces:**
- Consumes: `repair.Audit`, typed conflicts, `ComicService.Merge`, and atomic repository writes.
- Produces: versioned plan schema `1`.
- Apply selectors: exactly one of `--group-id keyless:5980` or `--all-approved`.
- Apply prerequisite: `--confirm-reviewed`, matching logical fingerprint, and a matching verified SQLite backup.

- [ ] **Step 1: Write planner and review-gate tests**

Cover:

```go
func TestBuildPlanNeverApprovesGroups(t *testing.T) {
	plan := BuildPlan(fixtureAudit(), time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC))
	if len(plan.Groups) == 0 {
		t.Fatal("expected groups")
	}
	for _, group := range plan.Groups {
		if group.Review.Approved || group.Review.Decision != DecisionReview {
			t.Fatalf("export auto-approved group %#v", group)
		}
	}
}

func TestChapterSignals(t *testing.T) {
	signals := ComputeSignals([]ComicSnapshot{
		{ID: 10, Titles: []string{"Hero"}, CurrentChap: 100, ComType: 3},
		{ID: 11, Titles: []string{"The hero"}, CurrentChap: 90, ComType: 3},
	})
	if signals.ChapterSpread != 10 || signals.ChapterRatio != 0.9 ||
		!signals.ChaptersClose || signals.ChaptersFar || !signals.ExactKey ||
		!signals.SameType || signals.NovelBoundary {
		t.Fatalf("unexpected signals: %#v", signals)
	}
}

func TestValidatePlanRejectsUnreviewedAndStaleApply(t *testing.T) {
	plan := BuildPlan(fixtureAudit(), time.Now().UTC())
	plan.Groups[0].Review = Review{
		Approved:   true,
		Decision:   DecisionMerge,
		Reviewer:   "",
		ReviewedAt: "",
	}
	if err := ValidatePlan(plan, fixtureAudit()); err == nil {
		t.Fatal("expected missing reviewer to fail")
	}
	stale := fixtureAudit()
	stale.DatabaseFingerprint = "stale"
	if err := ValidatePlan(BuildPlan(fixtureAudit(), time.Now().UTC()), stale); err == nil {
		t.Fatal("expected fingerprint mismatch")
	}
}
```

- [ ] **Step 2: Define the exact repair-plan schema**

Add:

```go
const PlanSchemaVersion = 1

type Decision string

const (
	DecisionReview      Decision = "review"
	DecisionRestoreKeys Decision = "restore_keys"
	DecisionMerge       Decision = "merge"
	DecisionBlock       Decision = "block"
)

type Review struct {
	Approved    bool     `json:"approved"`
	Decision    Decision `json:"decision"`
	Reviewer    string   `json:"reviewer"`
	ReviewedAt  string   `json:"reviewed_at"`
	CanonicalID int      `json:"canonical_id,omitempty"`
	MergeIDs    []int    `json:"merge_ids,omitempty"`
	Notes       string   `json:"notes"`
}

type Signals struct {
	ExactKey      bool    `json:"exact_key"`
	ExactTitle    bool    `json:"exact_title"`
	TitlePrefix   bool    `json:"title_prefix"`
	TitleTokens   bool    `json:"title_tokens"`
	SameType      bool    `json:"same_type"`
	NovelBoundary bool    `json:"novel_boundary"`
	ChaptersClose bool    `json:"chapters_close"`
	ChaptersFar   bool    `json:"chapters_far"`
	ChapterSpread int     `json:"chapter_spread"`
	ChapterRatio  float64 `json:"chapter_ratio"`
	GroupSize     int     `json:"group_size"`
}

type Group struct {
	GroupID             string                    `json:"group_id"`
	Reason              string                    `json:"reason"`
	Risk                string                    `json:"risk"`
	Keyless             ComicSnapshot             `json:"keyless"`
	Owners              []ComicSnapshot           `json:"owners"`
	ProposedKeys        []identity.ScopedTitleKey `json:"proposed_keys"`
	SuggestedCanonicalID int                       `json:"suggested_canonical_id,omitempty"`
	Signals             Signals                   `json:"signals"`
	Review              Review                    `json:"review"`
}

type Plan struct {
	SchemaVersion       int          `json:"schema_version"`
	GeneratedAt         string       `json:"generated_at"`
	DatabaseFingerprint string       `json:"database_fingerprint"`
	Summary             AuditSummary `json:"summary"`
	Groups              []Group      `json:"groups"`
}
```

- [ ] **Step 3: Port deterministic dedupe signals without auto-approval**

Use the historical thresholds already documented by this repository:

```go
const (
	maxLowRiskGroupSize = 4
	chapterSpreadClose  = 15
	chapterRatioClose   = 0.85
	chapterSpreadFar    = 50
	chapterRatioFar     = 0.50
)
```

Compute:

```go
spread := maxChapter - minChapter
ratio := 1.0
if maxChapter > 0 {
	ratio = float64(minChapter) / float64(maxChapter)
}
chaptersClose := spread <= chapterSpreadClose ||
	(maxChapter >= 20 && ratio >= chapterRatioClose)
chaptersFar := spread > chapterSpreadFar ||
	(maxChapter >= 30 && ratio < chapterRatioFar)
```

Round serialized `chapter_ratio` to four decimal places. Set:

- `blocked` for no proposed key or a novel/series boundary.
- `review` for group size above 4, far chapters, cross-type groups, or non-exact relationships.
- `low` only for same-type exact-key groups with close chapters and at most 4 members.

Risk is evidence only. Every group still exports:

```go
Review{
	Approved:   false,
	Decision:   DecisionReview,
	Reviewer:   "",
	ReviewedAt: "",
	Notes:      "",
}
```

Suggest the lowest comic ID only when no novel boundary exists. Never copy the suggestion into `Review.CanonicalID`.

- [ ] **Step 4: Validate every approved decision**

`ValidatePlan` must reject:

- schema version other than 1;
- logical fingerprint mismatch;
- duplicate group IDs;
- an approved group with empty reviewer or invalid RFC3339 `reviewed_at`;
- `review` or `block` marked approved;
- `restore_keys` when an active owner exists;
- `merge` with no owner, an unknown canonical ID, duplicate merge IDs, canonical in merge IDs, or any group member omitted;
- merge across a novel/series boundary;
- an unapproved group containing a mutation decision;
- comic snapshots whose IDs or proposed keys differ from the current audit.

`FilterApproved` returns one named approved group when `groupID` is non-empty, or every approved group when `allApproved` is true. It rejects both selectors together and rejects neither selector.

- [ ] **Step 5: Export and validate the versioned plan**

Add these query-only flags to `cmd/dedupe`:

```go
exportPlan := flag.String("export-plan", "", "write an unapproved discovery repair plan")
validatePlan := flag.String("validate-plan", "", "validate a reviewed plan without applying it")
```

`--export-plan` opens the database with `NewSQLiteComicRepositoryReadOnly`, calls `BuildPlan`, and writes the full plan with mode `0600`. Use a temporary sibling file, encode and indent JSON, call `Sync`, close the file, and rename it to the requested path; a failed export never truncates an existing plan.

`--validate-plan` opens the database read-only, decodes with `json.Decoder.DisallowUnknownFields`, re-audits, and calls `ValidatePlan(plan, audit)`. It prints:

```text
plan valid: groups=346 approved=0 merge=0 restore_keys=0 blocked=1
```

for the untouched live export. Validation never changes approval fields.

- [ ] **Step 6: Write failing offline-apply tests**

Create fixture tests that:

1. Export a plan.
2. Approve one `restore_keys` group with reviewer and timestamp.
3. Create a SQLite `.backup`.
4. Apply the group and verify the keyless count drops by one.
5. Restore the backup and verify the original logical fingerprint.
6. Inject a second-group conflict and verify the all-approved transaction rolls back the first group.
7. Reject a stale plan, missing backup, mismatched backup, missing confirmation, and an unapproved group.

Expected failure before implementation: `db_maint` recognizes only `--drop-legacy-identity-key`.

- [ ] **Step 7: Add guarded repair flags to db_maint**

Keep the legacy flag and add:

```go
applyPlan := flag.String("apply-discovery-plan", "", "apply a reviewed discovery repair plan")
backupPath := flag.String("backup", "", "verified pre-repair SQLite backup")
groupID := flag.String("group-id", "", "apply one approved group")
allApproved := flag.Bool("all-approved", false, "apply every explicitly approved group")
confirmReviewed := flag.Bool("confirm-reviewed", false, "confirm the plan received human review")
```

For repair:

1. Require the API and scheduler to be stopped as documented.
2. Require `--confirm-reviewed`.
3. Open the current database normally and the backup query-only.
4. Require both integrity checks to return `ok`.
5. Require current audit fingerprint, backup audit fingerprint, and plan fingerprint to match.
6. Re-audit current groups and call `ValidatePlan`.
7. Select one group or all approved groups.
8. Run all selected groups inside one `repo.WithTx`.
9. Re-audit after commit and print before/after counts.

Apply decisions inside the transaction:

```go
switch group.Review.Decision {
case repair.DecisionRestoreKeys:
	comic, err := txRepo.Get(ctx, group.Keyless.ID)
	if err != nil {
		return err
	}
	if _, err := txRepo.Update(ctx, comic); err != nil {
		return err
	}
case repair.DecisionMerge:
	service := usecase.NewComicServiceWithObserver(
		txRepo,
		nil,
		discovery.NoopObserver{},
	)
	for _, mergeID := range group.Review.MergeIDs {
		if _, err := service.Merge(ctx, group.Review.CanonicalID, mergeID); err != nil {
			return err
		}
	}
default:
	return fmt.Errorf("group %s has non-applicable decision %q", group.GroupID, group.Review.Decision)
}
```

Any error rolls back every selected group. There is no flag that approves groups or chooses a canonical row.

- [ ] **Step 8: Run planner and maintenance tests**

Run:

```bash
cd go_server
go test ./internal/discovery/repair ./cmd/dedupe ./cmd/db_maint -count=1
```

Expected: PASS, including the injected mid-batch rollback test.

- [ ] **Step 9: Commit**

```bash
git add go_server/internal/discovery/repair/plan.go \
  go_server/internal/discovery/repair/plan_test.go \
  go_server/cmd/dedupe/main.go \
  go_server/cmd/dedupe/main_test.go \
  go_server/cmd/db_maint/main.go \
  go_server/cmd/db_maint/main_test.go
git commit -m "feat: add reviewed discovery repair flow"
```

### Task 9: Add Make Targets, Rehearse, Roll Back, and Document Operations

**Files:**
- Modify: `Makefile:4-11,186-193,206-213`
- Modify: `go_server/cmd/dedupe/main_test.go`
- Create: `docs/runbooks/discovery-data-repair.md`

**Interfaces:**
- Root targets:
  - `db-audit-discovery`
  - `db-export-discovery-plan`
  - `db-validate-discovery-plan`
  - `db-apply-discovery-plan`
  - `db-audit-discovery-live`
- Inputs:
  - `DB_FILE`
  - `DISCOVERY_PLAN`
  - `DISCOVERY_BACKUP`
  - `DISCOVERY_GROUP`
  - `DISCOVERY_ALL_APPROVED`
  - `CONFIRM`

- [ ] **Step 1: Add failing Make target smoke assertions**

Extend the command tests:

```go
func TestMakeTargetsExposeDiscoveryWorkflow(t *testing.T) {
	command := exec.Command("make", "help")
	command.Dir = "../../.."
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make help failed: %v\n%s", err, output)
	}
	for _, target := range []string{
		"db-audit-discovery",
		"db-export-discovery-plan",
		"db-validate-discovery-plan",
		"db-apply-discovery-plan",
		"db-audit-discovery-live",
	} {
		if !strings.Contains(string(output), target) {
			t.Fatalf("make help missing %s:\n%s", target, output)
		}
	}
}
```

Add `os/exec` and `strings` to the test imports. Run:

```bash
cd go_server
go test ./cmd/dedupe -run MakeTargets -count=1
```

Expected: FAIL because the targets are absent.

- [ ] **Step 2: Add exact root forwarding targets**

Define:

```make
DISCOVERY_PLAN ?= db/recovery/discovery-repair-20261026.json
DISCOVERY_BACKUP ?=
DISCOVERY_GROUP ?=
DISCOVERY_ALL_APPROVED ?= 0
CONFIRM ?=
BACKUP_FILE ?=
```

Add:

```make
## @database:db-audit-discovery Read-only discovery integrity summary
db-audit-discovery:
	$(MAKE) -C go_server go-dedupe ARGS="--audit --db ../$(DB_FILE)"

## @database:db-export-discovery-plan Export an unapproved repair plan
db-export-discovery-plan:
	@mkdir -p "$$(dirname "$(DISCOVERY_PLAN)")"
	$(MAKE) -C go_server go-dedupe ARGS="--export-plan ../$(DISCOVERY_PLAN) --db ../$(DB_FILE)"

## @database:db-validate-discovery-plan Validate plan against current read-only audit
db-validate-discovery-plan:
	@test -f "$(DISCOVERY_PLAN)" || (echo "Plan not found: $(DISCOVERY_PLAN)"; exit 2)
	$(MAKE) -C go_server go-dedupe ARGS="--validate-plan ../$(DISCOVERY_PLAN) --db ../$(DB_FILE)"

## @database:db-apply-discovery-plan Apply reviewed repairs offline
db-apply-discovery-plan:
	@test "$(CONFIRM)" = "reviewed" || (echo "Set CONFIRM=reviewed"; exit 2)
	@test -n "$(DISCOVERY_BACKUP)" || (echo "Set DISCOVERY_BACKUP"; exit 2)
	@test -n "$(DISCOVERY_GROUP)" -o "$(DISCOVERY_ALL_APPROVED)" = "1" || (echo "Set DISCOVERY_GROUP or DISCOVERY_ALL_APPROVED=1"; exit 2)
	@selector=""; \
	if [ -n "$(DISCOVERY_GROUP)" ]; then selector="--group-id $(DISCOVERY_GROUP)"; fi; \
	if [ "$(DISCOVERY_ALL_APPROVED)" = "1" ]; then selector="--all-approved"; fi; \
	COMICS_SQLITE_PATH="../$(DB_FILE)" $(MAKE) -C go_server go-db-maint \
	  ARGS="--apply-discovery-plan ../$(DISCOVERY_PLAN) --backup ../$(DISCOVERY_BACKUP) --confirm-reviewed $$selector"

## @database:db-audit-discovery-live Run opt-in no-write audit test
db-audit-discovery-live:
	cd go_server && AUDIT_DISCOVERY=1 COMICS_SQLITE_PATH=../$(DB_FILE) \
	  go test ./cmd/dedupe -run TestAuditLiveDiscoveryIntegrity -count=1 -v
```

Add all five root names to `.PHONY`. They deliberately forward through the existing `go-dedupe` and `go-db-maint` targets; Task 3 already added the only new Go-server target.

Also change the existing root `go-dedupe` help text from “Audit/merge” to
“Audit/export/validate discovery repair plans”; mutation now belongs only to
the guarded `go-db-maint` apply path.

- [ ] **Step 3: Make backup and restore safe for the repair rehearsal**

Change `db-backup` to:

```make
db-backup:
	@mkdir -p db/backups
	@ts=$$(date +%Y%m%d_%H%M%S); \
	dest="$(BACKUP_FILE)"; \
	if [ -z "$$dest" ]; then dest="db/backups/comics_backup_$${ts}.db"; fi; \
	if [ -e "$$dest" ]; then echo "Backup already exists: $$dest"; exit 2; fi; \
	sqlite3 "$(DB_FILE)" ".backup '$${dest}'"; \
	result=$$(sqlite3 -readonly "$$dest" "PRAGMA integrity_check;"); \
	test "$$result" = "ok"; \
	echo "Backup: $${dest} (integrity=ok)"
```

This preserves timestamped default behavior and lets rehearsal use a known
path. Workstream 3 Task 7 later strengthens this same target and owns its final
implementation; that version must retain `BACKUP_FILE`, the no-overwrite rule,
and backup integrity verification rather than adding a second backup target.

Strengthen `db-restore` in this workstream as well, so repair rollback is safe
before Workstream 3 lands:

```make
db-restore:
	@test "$(CONFIRM_OFFLINE)" = "yes" || (echo "Set CONFIRM_OFFLINE=yes after stopping the API" && exit 2)
	@test -n "$(BACKUP)" || (echo "Set BACKUP=db/backups/<file>.db" && exit 2)
	@test -f "$(BACKUP)" || (echo "Backup not found: $(BACKUP)" && exit 1)
	@result=$$(sqlite3 -readonly "$(BACKUP)" "PRAGMA integrity_check;"); \
		test "$$result" = "ok" || (echo "Backup integrity failed: $$result" && exit 1)
	@trap 'rm -f "$(DB_FILE).restore"' EXIT; \
		cp "$(BACKUP)" "$(DB_FILE).restore"; \
		rm -f "$(DB_FILE)-wal" "$(DB_FILE)-shm"; \
		mv "$(DB_FILE).restore" "$(DB_FILE)"; \
		echo "Restored $(DB_FILE) from $(BACKUP)"
```

Workstream 3 Task 7 retains this offline confirmation, backup verification,
sidecar cleanup, and atomic rename; it does not introduce a second restore
contract.

- [ ] **Step 4: Write the operational runbook**

Document these gates verbatim:

- stop API and scheduler before backup or apply;
- audit/export/validate are read-only;
- inspect every conflict's titles, types, chapters, tracking, publishers, and metadata;
- never approve across a novel/series boundary;
- far chapters (`spread > 50` or `max >= 30` and `ratio < 0.50`) require review regardless of title;
- set reviewer, RFC3339 review time, decision, canonical ID, merge IDs, and notes;
- apply a canary group first on a rehearsal copy;
- use `--all-approved` only after every approved group has recorded review metadata;
- restore the backup if post-apply audit or smoke tests fail;
- every restore requires the API/scheduler stopped and
  `CONFIRM_OFFLINE=yes`;
- repair SQLite first, then run the guarded PostgreSQL import; do not repair the two catalogs independently.

- [ ] **Step 5: Rehearse one non-conflicting repair against a copy**

Run from repository root with the API stopped:

```bash
rm -f db/backups/discovery_rehearsal_source.db \
  db/backups/discovery_rehearsal_before_apply.db \
  db/recovery/discovery-rehearsal.db \
  db/recovery/discovery-rehearsal-plan.json \
  db/recovery/discovery-rehearsal-plan.reviewed.json
make db-backup \
  DB_FILE=db/comics.db \
  BACKUP_FILE=db/backups/discovery_rehearsal_source.db
cp db/backups/discovery_rehearsal_source.db db/recovery/discovery-rehearsal.db
make db-backup \
  DB_FILE=db/recovery/discovery-rehearsal.db \
  BACKUP_FILE=db/backups/discovery_rehearsal_before_apply.db
make db-export-discovery-plan \
  DB_FILE=db/recovery/discovery-rehearsal.db \
  DISCOVERY_PLAN=db/recovery/discovery-rehearsal-plan.json
jq '(.groups[] | select(.group_id == "keyless:5980") | .review) = {
  "approved": true,
  "decision": "restore_keys",
  "reviewer": "architecture-review-rehearsal",
  "reviewed_at": "2026-10-26T15:00:00Z",
  "notes": "CJK-only title receives Unicode fallback key"
}' db/recovery/discovery-rehearsal-plan.json \
  > db/recovery/discovery-rehearsal-plan.reviewed.json
make db-validate-discovery-plan \
  DB_FILE=db/recovery/discovery-rehearsal.db \
  DISCOVERY_PLAN=db/recovery/discovery-rehearsal-plan.reviewed.json
make db-apply-discovery-plan \
  DB_FILE=db/recovery/discovery-rehearsal.db \
  DISCOVERY_PLAN=db/recovery/discovery-rehearsal-plan.reviewed.json \
  DISCOVERY_BACKUP=db/backups/discovery_rehearsal_before_apply.db \
  DISCOVERY_GROUP=keyless:5980 \
  CONFIRM=reviewed
make db-audit-discovery DB_FILE=db/recovery/discovery-rehearsal.db
```

Expected progression:

```text
before: keyless=346 conflicts=341 restorable=4 blocked=1
applied: groups=1 merges=0 restored_keys=1
after: keyless=345 conflicts=341 restorable=3 blocked=1
```

- [ ] **Step 6: Prove post-commit rollback from the backup**

Run:

```bash
make db-restore \
  DB_FILE=db/recovery/discovery-rehearsal.db \
  BACKUP=db/backups/discovery_rehearsal_before_apply.db \
  CONFIRM_OFFLINE=yes
make db-check DB_FILE=db/recovery/discovery-rehearsal.db
make db-audit-discovery DB_FILE=db/recovery/discovery-rehearsal.db
```

Expected:

```text
db/recovery/discovery-rehearsal.db: ok
discovery audit: total=31309 active=31308 keys=31892 active_owners=30962 keyless=346 conflicts=341 restorable=4 blocked=1 deleted_keys=0 orphan_keys=0
```

Delete only the rehearsal artifacts after recording the command output. Do not delete the verified production backup.

- [ ] **Step 7: Run the complete verification set**

Run:

```bash
make go-test
make contract-test
make go-vet
make go-build
make db-audit-discovery-live
```

Expected:

```text
Go tests: PASS
HTTP contract tests: PASS
go vet: PASS
server and importer builds: PASS
live audit: PASS with unchanged DB/WAL/SHM digest set
```

If a local PostgreSQL test database is available, also run:

```bash
cd go_server
COMICS_POSTGRES_URL='postgres://postgres:postgres@localhost:55432/comics_test?sslmode=disable' \
  make go-test-postgres-discovery
```

Expected: all PostgreSQL discovery contract subtests PASS.

- [ ] **Step 8: Commit**

```bash
git add Makefile go_server/cmd/dedupe/main_test.go docs/runbooks/discovery-data-repair.md
git commit -m "docs: add discovery repair runbook and targets"
```

## Production Repair Sequence

After all nine implementation tasks and code review:

1. Stop the API and periodic scheduler.
2. Run `make db-check`.
3. Create a fresh backup with `make db-backup BACKUP_FILE=db/backups/discovery_pre_repair.db`; fail if that path already exists.
4. Export `db/recovery/discovery-repair-20261026.json`.
5. Review all 346 groups. The 341 conflicts require merge/separate-series investigation; the four CJK records may use `restore_keys`; comic 4561 remains blocked until title correction or deletion.
6. Validate the reviewed plan against the unchanged database.
7. Rehearse the exact reviewed plan on a backup copy.
8. Apply one approved canary with `DISCOVERY_GROUP`, re-export if continuing from the changed database, or restore and apply all approved groups in one reviewed transaction.
9. Require post-apply `integrity_check=ok`, `keyless=0`, `deleted_keys=0`, and `orphan_keys=0` before restart. If comic 4561 remains active and untitled, this gate intentionally fails.
10. Start the API, run one authorized create/update/delete smoke cycle, and verify the conflict and ambiguity metrics are present.
11. If PostgreSQL is used, import only from the repaired SQLite source and retain both generated PostgreSQL backup tables until verification completes.

## Rollback Rules

- Before commit: any typed conflict, stale fingerprint, invalid review, SQL error, or post-transaction invariant failure rolls back the complete selected batch.
- After SQLite commit: stop the API, run `make db-restore DB_FILE=db/comics.db BACKUP=db/backups/discovery_pre_repair.db CONFIRM_OFFLINE=yes`, then run `make db-check` and `make db-audit-discovery`.
- After PostgreSQL import: restore both timestamped `comics_backup_*` and `comic_title_keys_backup_*` tables in one PostgreSQL transaction, reset `comic_id_seq`, and rerun `make repo-contract-test` after Workstream 3; before that workstream, use the interim `go-test-postgres-discovery` target.
- Never repair by deleting rows directly from `comic_title_keys`; use the reviewed plan so comic merges, normalization, keys, and rollback remain coordinated.

## Completion Criteria

- Read-only audit/export reports all 346 baseline keyless active comics and does not change database, WAL, or SHM state.
- Export reports 341 active conflicts, four restorable Unicode-only records, and one blocked empty-title record.
- Every exported group is unapproved and requires reviewer identity and RFC3339 review time.
- SQLite and PostgreSQL contract tests prove typed conflicts, self exclusion, atomic rollback, and deleted-key release.
- Create, update, merge, scrape, direct repository writes, and PostgreSQL import normalize titles consistently.
- Resolver ties and near-ties return `ErrAmbiguousComicMatch`; track, chapter, and ID do not choose a winner.
- `/metrics` exposes bounded resolution and conflict counters.
- HTTP mutations return stable 409 codes for key conflict and resolver ambiguity.
- Reviewed repair supports a canary or all explicitly approved groups, with no auto-approval path.
- Backup fingerprint, plan fingerprint, and current database fingerprint must match before apply.
- Rehearsal demonstrates apply, audit, and restore before production data changes.
