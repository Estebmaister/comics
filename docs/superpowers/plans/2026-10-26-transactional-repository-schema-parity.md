# Transactional Repository and Schema Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every catalog row/title-key write atomic, give SQLite and PostgreSQL one authoritative migration path and one behavioral repository contract, and provide a rehearsable SQLite foreign-key migration with verified backup and rollback.

**Architecture:** Keep `domain.ComicRepository` as the application boundary and keep `SQLComicRepository` as the one production REST adapter for both dialects. Versioned, embedded, dialect-specific SQL under one migration package becomes the only schema source; API startup validates the schema but never performs the offline SQLite rebuild. SQLite remains the default single-instance catalog, while PostgreSQL remains an optional adapter and import target.

**Tech Stack:** Go 1.27.3, `database/sql`, `modernc.org/sqlite`, `pgx/v5`, `golang-migrate/v4`, `testcontainers-go`, SQLite, PostgreSQL, Make, GitHub Actions.

## Global Constraints

- SQLite remains the default catalog store; MongoDB remains the user/auth store; PostgreSQL remains optional.
- SQLite mode runs exactly one API replica and one periodic scheduler.
- Public catalog reads and admin mutation policy are owned by workstream 1 and are not changed here.
- Every active comic must have at least one valid discovery key before the live SQLite foreign-key migration is allowed to run.
- One `(scope, title_key)` has exactly one owner; a conflict is typed and never silently discarded.
- Comic rows and discovery-key changes commit or roll back together for REST, scrape, merge, and import writes.
- Deleted comics do not retain discovery keys.
- No schema-changing migration runs automatically during API startup.
- The gRPC repository remains non-production; this work removes its private migration path but does not claim gRPC behavior parity.
- Never run a destructive command against `db/comics.db` without a verified SQLite backup and an offline maintenance window.

---

## Dependency Gate and Verified Baseline

This plan depends on
`docs/superpowers/plans/2026-10-26-discovery-data-integrity.md`.
Land and verify that plan's source changes before Task 1 because both plans
modify repository, importer, tests, and Makefiles. The migration package can be
prototyped in an isolated branch while review is pending, but it must be rebased
onto the discovery work before any workstream-3 commit. Complete the approved
live-data repair before applying the live migration in Task 7.

The dependency must provide this exact error contract:

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

It must also make `SyncDiscoveryKeys` use strict inserts, normalize every write,
exclude the comic being updated from duplicate detection, and remove keys from
deleted comics. Its Tasks 2, 5, and 6 are the source of truth for atomic
repository writes, transactional REST resolution, and normalized one-transaction
PostgreSQL import. This plan adds final-schema regression contracts and the
canonical migration call; it does not reimplement those dependency tasks. The
live-data gate immediately before Task 7 is:

```bash
make db-check
make db-audit-discovery DB_FILE=db/comics.db
```

Expected:

```text
db/comics.db: ok
discovery summary includes keyless=0
discovery summary includes conflicts=0
discovery summary includes blocked=0
discovery summary includes orphan_keys=0
discovery summary includes deleted_key_rows=0
```

Do not infer that the current database already satisfies that gate. Read-only
inspection on 2026-10-26 established this starting point:

```text
PRAGMA foreign_keys = 0
comic_title_keys foreign-key rows = 0
active_keyless = 346
orphan_keys = 0
deleted_key_rows = 0
```

The focused repository/import/use-case baseline was green:

```bash
cd go_server
go test ./internal/repo/comics ./cmd/import_comics_postgres ./internal/usecase
```

Expected before implementation:

```text
Go test: 18 passed in 3 packages
```

## File Structure

Create:

- `go_server/internal/repo/migrations/migrations.go` — embedded migration runner, current-version validation, and dialect selection.
- `go_server/internal/repo/migrations/migrations_test.go` — empty and legacy SQLite migration/rollback tests.
- `go_server/internal/repo/migrations/backup.go` — consistent SQLite backup and integrity verification.
- `go_server/internal/repo/migrations/sqlite/000001_catalog_baseline.up.sql`
- `go_server/internal/repo/migrations/sqlite/000001_catalog_baseline.down.sql`
- `go_server/internal/repo/migrations/sqlite/000002_title_keys_foreign_key.up.sql`
- `go_server/internal/repo/migrations/sqlite/000002_title_keys_foreign_key.down.sql`
- `go_server/internal/repo/migrations/postgres/000001_catalog_baseline.up.sql`
- `go_server/internal/repo/migrations/postgres/000001_catalog_baseline.down.sql`
- `go_server/internal/repo/migrations/postgres/000002_title_keys_foreign_key.up.sql`
- `go_server/internal/repo/migrations/postgres/000002_title_keys_foreign_key.down.sql`
- `go_server/internal/repo/migrations/testdata/sqlite_current_legacy.sql` — reviewed pre-migration production SQLite fixture.
- `go_server/cmd/migrate/main_test.go` — command parsing, backup, up, bounded down, and failure-safety tests.
- `go_server/internal/repo/comics/repository_contract_test.go` — the shared dialect-neutral contract.
- `go_server/internal/repo/comics/repository_contract_postgres_test.go` — PostgreSQL container harness behind the `integration` build tag.
- `go_server/cmd/import_comics_postgres/integration_test.go` — end-to-end atomic import tests behind the `integration` build tag.

Modify:

- `go_server/domain/comic.go` — document transactional semantics without changing method signatures.
- `go_server/internal/repo/comics/sql_repository.go` — remove inline schema creation while preserving the dependency's atomic write wrappers.
- `go_server/internal/repo/comics/title_keys.go` — retain strict transactional key behavior from workstream 2 and remove residual runtime DDL/backfill.
- `go_server/internal/repo/comics/sqlite_health.go` — verify that foreign-key enforcement is enabled and that `foreign_key_check` is clean.
- `go_server/internal/repo/comics/sql_repository_test.go` — migrate fixtures and add atomicity/FK assertions.
- `go_server/internal/repo/comics/sql_repository_identity_test.go` — assert rollback after typed key conflicts.
- `go_server/internal/repo/comics/discovery_audit_test.go` — keep legacy read-only audit usable before migration.
- `go_server/internal/usecase/comics_test.go` — prove REST create/update use one transaction.
- `go_server/api/route/comics_test.go` — construct schema through the migration package.
- `go_server/internal/scrape/scrape_test.go` — construct schema through the migration package.
- `go_server/internal/repo/comics_repo.go` — remove the legacy gRPC repository's private migration source and startup migrator.
- `go_server/cmd/migrate/main.go` — replace the broken PostgreSQL-only command with the shared SQLite-default CLI.
- `go_server/cmd/import_comics_postgres/main.go` — run canonical migrations before the dependency's prepared atomic replacement.
- `go_server/cmd/import_comics_postgres/main_test.go` — retain normalized preflight tests and add migration/commit output checks.
- `go_server/cmd/db_maint/main.go`, `go_server/cmd/db_maint/main_test.go` — remove the legacy schema-changing identity-column flag while retaining reviewed data repair.
- `go_server/sqlc.yml` — read schema from the canonical migration directories.
- `go_server/internal/repo/sql/query/comics_sqlite.sql`, `go_server/internal/repo/sql/query/comics_postgre.sql` — keep dormant sqlc queries valid against the canonical serialized-text schema.
- `go_server/internal/repo/sql/sqlite/`, `go_server/internal/repo/sql/postgres/` — regenerate checked-in sqlc outputs from the canonical schema.
- `go_server/go.mod`, `go_server/go.sum` — add the PostgreSQL testcontainers module.
- `go_server/Makefile`, `Makefile` — expose safe migration, migration-test, repository-contract, backup, and rehearsal targets.
- `.github/workflows/ci.yml` — run the two-dialect repository contract.
- `go_server/README.md` — document migration, import, backup, and restore operations.
- `docs/runbooks/discovery-data-repair.md` — carry the strengthened offline restore confirmation into the dependency runbook.

Delete:

- `go_server/internal/repo/sql/migrations/001_initial_schema.sql`
- `go_server/internal/repo/sql/migrations/002_add_triggers.sql`
- `go_server/internal/repo/sql/migrations/003_add_cover_visible.sql`
- `go_server/internal/repo/sql/migrations/004_comic_title_keys.sql`
- `go_server/internal/repo/sql/migrations/005_drop_comics_identity_key.sql`
- `go_server/internal/repo/sql/schema/comics_sqlite.sql`
- `go_server/internal/repo/sql/schema/comics_postgre.sql`
- `go_server/internal/repo/comics/sql_repository_postgres_test.go`

The generated `internal/repo/sql/sqlite/` and `internal/repo/sql/postgres/`
packages remain generated outputs. Workstream 8 decides whether the non-REST
gRPC/sqlc surface is retired.

## Exact Interfaces

The migration package owns this API:

```go
package migrations

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
	CurrentVersion  uint    = 2
)

var ErrSchemaNotCurrent = errors.New("catalog schema is not current")

type State struct {
	Version uint
	Dirty   bool
}

// Open takes ownership of db; callers must not close db after calling Open.
func Open(db *sql.DB, dialect Dialect) (*Runner, error)
func (r *Runner) Up(ctx context.Context) (State, error)
func (r *Runner) Down(ctx context.Context, steps uint) (State, error)
func (r *Runner) State() (State, error)
func (r *Runner) Close() error

// ReadState is read-only and returns State{} when the migration table is absent.
func ReadState(ctx context.Context, db *sql.DB, dialect Dialect) (State, error)
func ValidateCurrent(ctx context.Context, db *sql.DB, dialect Dialect) error
func SQLiteDSN(path string) string
func SQLiteReadOnlyDSN(path string) string
func BackupSQLite(ctx context.Context, dbPath string, backupDir string) (string, error)
func VerifySQLite(ctx context.Context, dbPath string) error
```

`domain.ComicRepository` keeps its existing signatures. Its comments become
part of the contract:

```go
type ComicRepository interface {
	// Create atomically persists the comic and its discovery keys.
	Create(ctx context.Context, comic Comic) (Comic, error)
	// Update atomically persists the comic and replaces its discovery keys.
	Update(ctx context.Context, comic Comic) (Comic, error)
	// Delete removes the comic; schema-level ON DELETE CASCADE removes its keys.
	Delete(ctx context.Context, id int) error
	// WithTx commits fn as one transaction. Nested calls reuse the active transaction.
	WithTx(ctx context.Context, fn func(ComicRepository) error) error

	// Existing read and key methods remain unchanged.
}
```

The PostgreSQL importer keeps the command-local
`prepareImportComics([]sourceComic) ([]preparedComic, error)` and
extends the discovery plan's replacement interface to
`replaceTargetComics(context.Context, *sql.DB, []preparedComic) (verifyCounts, importBackup, error)`.
No bulk-import method is added to the general repository interface.

### Task 1: Replace Schema Drift with One Embedded Migration Source

**Files:**

- Create: `go_server/internal/repo/migrations/migrations.go`
- Create: `go_server/internal/repo/migrations/migrations_test.go`
- Create: `go_server/internal/repo/migrations/testdata/sqlite_current_legacy.sql`
- Create: all eight SQL files under `go_server/internal/repo/migrations/{sqlite,postgres}/`
- Modify: `go_server/sqlc.yml`
- Modify: `go_server/internal/repo/sql/query/comics_sqlite.sql`
- Modify: `go_server/internal/repo/sql/query/comics_postgre.sql`
- Regenerate: `go_server/internal/repo/sql/sqlite/*.go`
- Regenerate: `go_server/internal/repo/sql/postgres/*.go`
- Delete: `go_server/internal/repo/sql/migrations/*.sql`
- Delete: `go_server/internal/repo/sql/schema/comics_sqlite.sql`
- Delete: `go_server/internal/repo/sql/schema/comics_postgre.sql`

**Interfaces:**

- Consumes: `*sql.DB`, `migrations.Dialect`.
- Produces: `Open`, `Runner.Up`, `Runner.Down`, `Runner.State`,
  read-only `ReadState`, `ValidateCurrent`, and embedded version `2` schemas.

- [ ] **Step 1: Write the failing legacy-SQLite migration tests**

Add tests that create the reviewed pre-migration shape: `comics`,
`comic_title_keys` without a foreign key, one comic, and one key. The assertions
must be exact:

```go
func TestSQLiteLegacyMigrationAddsCascadeAndPreservesRows(t *testing.T) {
	dbPath := createLegacySQLiteFixture(t)
	db := openSQLiteForMigration(t, dbPath)
	runner, err := Open(db, DialectSQLite)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runner.Close()) })

	state, err := runner.Up(context.Background())
	require.NoError(t, err)
	require.Equal(t, State{Version: CurrentVersion, Dirty: false}, state)

	check := openSQLiteReadOnly(t, dbPath)
	defer check.Close()
	var comics, keys int
	require.NoError(t, check.QueryRow(`SELECT COUNT(*) FROM comics`).Scan(&comics))
	require.NoError(t, check.QueryRow(`SELECT COUNT(*) FROM comic_title_keys`).Scan(&keys))
	require.Equal(t, 1, comics)
	require.Equal(t, 1, keys)
	requireForeignKey(t, check, "comic_title_keys", "comic_id", "comics", "id", "CASCADE")
}

func TestSQLiteLegacyMigrationFailureRollsBackTableRebuild(t *testing.T) {
	dbPath := createLegacySQLiteFixtureWithOrphanKey(t)
	runner := openRunner(t, dbPath, DialectSQLite)
	_, err := runner.Up(context.Background())
	require.Error(t, err)

	check := openSQLiteReadOnly(t, dbPath)
	defer check.Close()
	requireNoForeignKeys(t, check, "comic_title_keys")
	requireTableAbsent(t, check, "comic_title_keys__with_fk")
	requireRowCounts(t, check, 1, 2)
}

func TestSQLiteMigrationDownOneStepPreservesRows(t *testing.T) {
	dbPath := createLegacySQLiteFixture(t)
	runner := openRunner(t, dbPath, DialectSQLite)
	_, err := runner.Up(context.Background())
	require.NoError(t, err)

	state, err := runner.Down(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, State{Version: 1, Dirty: false}, state)

	check := openSQLiteReadOnly(t, dbPath)
	defer check.Close()
	requireNoForeignKeys(t, check, "comic_title_keys")
	requireRowCounts(t, check, 1, 1)
}
```

`createLegacySQLiteFixture` must use a checked-in DDL fixture copied from the
current production SQLite shape, not the reduced DDL in
`sql_repository_test.go`, so migration tests detect column loss. Assert with
`PRAGMA table_info(comics)` that the fixture contains exactly `id`, `titles`,
`current_chap`, `cover`, `last_update`, `com_type`, `status`, `published_in`,
`genres`, `description`, `author`, `track`, `viewed_chap`, `rating`, `deleted`,
and `cover_visible` before running the migration.

- [ ] **Step 2: Run the migration tests and verify they fail**

Run:

```bash
cd go_server
go test ./internal/repo/migrations -run 'TestSQLite(LegacyMigration|MigrationDown)' -count=1
```

Expected: build failure because package `internal/repo/migrations` and `Open`
do not exist.

- [ ] **Step 3: Add the embedded runner**

Implement `migrations.go` around `source/iofs`, the modernc-compatible
`database/sqlite` driver, and `database/pgx/v5`. Use
`catalog_schema_migrations` so an old, incompatible `schema_migrations` table
cannot be mistaken for the canonical version table.

```go
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"

	gmigrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	sqlitedriver "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sqlite/*.sql postgres/*.sql
var files embed.FS

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
	CurrentVersion  uint    = 2
	migrationsTable         = "catalog_schema_migrations"
)

var ErrSchemaNotCurrent = errors.New("catalog schema is not current")

type State struct {
	Version uint
	Dirty   bool
}

type Runner struct {
	m *gmigrate.Migrate
}

func Open(db *sql.DB, dialect Dialect) (runner *Runner, err error) {
	if db == nil {
		return nil, errors.New("catalog database is required")
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}()

	source, err := iofs.New(files, string(dialect))
	if err != nil {
		return nil, fmt.Errorf("open %s migration source: %w", dialect, err)
	}

	var driver database.Driver
	switch dialect {
	case DialectSQLite:
		driver, err = sqlitedriver.WithInstance(db, &sqlitedriver.Config{
			MigrationsTable: migrationsTable,
			NoTxWrap:       false,
		})
	case DialectPostgres:
		driver, err = pgxdriver.WithInstance(db, &pgxdriver.Config{
			MigrationsTable: migrationsTable,
		})
	default:
		_ = source.Close()
		return nil, fmt.Errorf("unsupported catalog dialect %q", dialect)
	}
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("open %s migration database: %w", dialect, err)
	}

	m, err := gmigrate.NewWithInstance("iofs", source, string(dialect), driver)
	if err != nil {
		_ = source.Close()
		_ = driver.Close()
		return nil, err
	}
	return &Runner{m: m}, nil
}

func (r *Runner) Up(ctx context.Context) (State, error) {
	if err := r.run(ctx, r.m.Up); err != nil {
		return State{}, err
	}
	return r.State()
}

func (r *Runner) Down(ctx context.Context, steps uint) (State, error) {
	if steps == 0 {
		return State{}, errors.New("down requires at least one step")
	}
	if err := r.run(ctx, func() error { return r.m.Steps(-int(steps)) }); err != nil {
		return State{}, err
	}
	return r.State()
}

func (r *Runner) run(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			select {
			case r.m.GracefulStop <- true:
			case <-done:
			}
		case <-done:
		}
	}()
	err := fn()
	if errors.Is(err, gmigrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}

func (r *Runner) State() (State, error) {
	version, dirty, err := r.m.Version()
	if errors.Is(err, gmigrate.ErrNilVersion) {
		return State{}, nil
	}
	return State{Version: version, Dirty: dirty}, err
}

func (r *Runner) Close() error {
	sourceErr, databaseErr := r.m.Close()
	return errors.Join(sourceErr, databaseErr)
}

func SQLiteDSN(path string) string {
	u := &url.URL{Scheme: "file", Path: path}
	query := u.Query()
	query.Add("_pragma", "journal_mode=WAL")
	query.Add("_pragma", "busy_timeout=30000")
	query.Add("_pragma", "foreign_keys=ON")
	u.RawQuery = query.Encode()
	return u.String()
}

func SQLiteReadOnlyDSN(path string) string {
	u := &url.URL{Scheme: "file", Path: path}
	query := u.Query()
	query.Set("mode", "ro")
	query.Add("_pragma", "busy_timeout=30000")
	query.Add("_pragma", "foreign_keys=ON")
	u.RawQuery = query.Encode()
	return u.String()
}

func ReadState(ctx context.Context, db *sql.DB, dialect Dialect) (State, error) {
	var exists bool
	var existsQuery string
	switch dialect {
	case DialectSQLite:
		existsQuery = `SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type = 'table' AND name = 'catalog_schema_migrations'
		)`
	case DialectPostgres:
		existsQuery = `SELECT to_regclass('public.catalog_schema_migrations') IS NOT NULL`
	default:
		return State{}, fmt.Errorf("unsupported catalog dialect %q", dialect)
	}
	if err := db.QueryRowContext(ctx, existsQuery).Scan(&exists); err != nil {
		return State{}, err
	}
	if !exists {
		return State{}, nil
	}
	var state State
	err := db.QueryRowContext(
		ctx,
		`SELECT version, dirty FROM catalog_schema_migrations LIMIT 1`,
	).Scan(&state.Version, &state.Dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, nil
	}
	return state, err
}

func ValidateCurrent(ctx context.Context, db *sql.DB, dialect Dialect) error {
	state, err := ReadState(ctx, db, dialect)
	if err != nil {
		return fmt.Errorf("%w: read %s migration state: %v", ErrSchemaNotCurrent, dialect, err)
	}
	if state.Dirty || state.Version != CurrentVersion {
		return fmt.Errorf(
			"%w: %s version=%d dirty=%t, want version=%d dirty=false; run make go-migrate-up before starting the API",
			ErrSchemaNotCurrent,
			dialect,
			state.Version,
			state.Dirty,
			CurrentVersion,
		)
	}
	return nil
}
```

`SQLiteDSN` is shared by migration, backup, test-fixture, and runtime repository
connections so the writable per-connection pragma policy cannot drift.
`SQLiteReadOnlyDSN` is used by version inspection, backup verification, and
read-only discovery audit; it never requests a journal-mode change.

- [ ] **Step 4: Add the canonical SQLite migrations**

`000001_catalog_baseline.up.sql` must contain the complete logical schema:

```sql
CREATE TABLE IF NOT EXISTS comics (
  id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  titles        TEXT    NOT NULL,
  current_chap  INTEGER NOT NULL DEFAULT 0,
  cover         TEXT    NOT NULL DEFAULT '',
  last_update   INTEGER NOT NULL DEFAULT 0,
  com_type      INTEGER NOT NULL DEFAULT 0,
  status        INTEGER NOT NULL DEFAULT 0,
  published_in  TEXT    NOT NULL DEFAULT '',
  genres        TEXT    NOT NULL DEFAULT '',
  description   TEXT    NOT NULL DEFAULT '',
  author        TEXT    NOT NULL DEFAULT '',
  track         INTEGER NOT NULL DEFAULT 0,
  viewed_chap   INTEGER NOT NULL DEFAULT 0,
  rating        INTEGER NOT NULL DEFAULT 0,
  deleted       INTEGER NOT NULL DEFAULT 0,
  cover_visible INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS comic_title_keys (
  comic_id  INTEGER NOT NULL,
  scope     TEXT    NOT NULL,
  title_key TEXT    NOT NULL,
  PRIMARY KEY (scope, title_key)
);

CREATE INDEX IF NOT EXISTS idx_comic_title_keys_comic_id
  ON comic_title_keys(comic_id);
CREATE INDEX IF NOT EXISTS idx_comics_active_update
  ON comics(deleted, last_update DESC, id);
CREATE INDEX IF NOT EXISTS idx_comics_active_track_update
  ON comics(deleted, track, last_update DESC, id);
```

Its down file drops `comic_title_keys` first and then `comics`.

`000002_title_keys_foreign_key.up.sql` performs the offline child-table rebuild:

```sql
CREATE TABLE comic_title_keys__with_fk (
  comic_id  INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
  scope     TEXT    NOT NULL,
  title_key TEXT    NOT NULL,
  PRIMARY KEY (scope, title_key)
);

INSERT INTO comic_title_keys__with_fk (comic_id, scope, title_key)
SELECT comic_id, scope, title_key
FROM comic_title_keys;

DROP TABLE comic_title_keys;
ALTER TABLE comic_title_keys__with_fk RENAME TO comic_title_keys;

CREATE INDEX idx_comic_title_keys_comic_id
  ON comic_title_keys(comic_id);
```

Its down file performs the inverse table rebuild without the `REFERENCES`
clause. Both directions preserve all key rows.

- [ ] **Step 5: Add the canonical PostgreSQL migrations**

The baseline uses the same column names and serialized text representation as
the REST repository:

```sql
CREATE SEQUENCE IF NOT EXISTS comic_id_seq;

CREATE TABLE IF NOT EXISTS comics (
  id            INTEGER PRIMARY KEY DEFAULT nextval('comic_id_seq'),
  titles        TEXT    NOT NULL,
  current_chap  INTEGER NOT NULL DEFAULT 0,
  cover         TEXT    NOT NULL DEFAULT '',
  last_update   BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM now())::BIGINT,
  com_type      INTEGER NOT NULL DEFAULT 0,
  status        INTEGER NOT NULL DEFAULT 0,
  published_in  TEXT    NOT NULL DEFAULT '',
  genres        TEXT    NOT NULL DEFAULT '',
  description   TEXT    NOT NULL DEFAULT '',
  author        TEXT    NOT NULL DEFAULT '',
  track         BOOLEAN NOT NULL DEFAULT false,
  viewed_chap   INTEGER NOT NULL DEFAULT 0,
  rating        INTEGER NOT NULL DEFAULT 0,
  deleted       BOOLEAN NOT NULL DEFAULT false,
  cover_visible BOOLEAN NOT NULL DEFAULT true
);

ALTER SEQUENCE comic_id_seq OWNED BY comics.id;
ALTER TABLE comics ALTER COLUMN id SET DEFAULT nextval('comic_id_seq');

DROP TRIGGER IF EXISTS update_comics_last_update ON comics;
DROP FUNCTION IF EXISTS update_last_update();

CREATE TABLE IF NOT EXISTS comic_title_keys (
  comic_id  INTEGER NOT NULL,
  scope     TEXT    NOT NULL,
  title_key TEXT    NOT NULL,
  PRIMARY KEY (scope, title_key)
);

CREATE INDEX IF NOT EXISTS idx_comic_title_keys_comic_id
  ON comic_title_keys(comic_id);
CREATE INDEX IF NOT EXISTS idx_comics_active_update
  ON comics(deleted, last_update DESC, id);
CREATE INDEX IF NOT EXISTS idx_comics_active_track_update
  ON comics(deleted, track, last_update DESC, id);
```

The trigger/function removal adopts databases touched by the retired migration
set: that trigger writes a timestamp into the repository's `BIGINT`
`last_update` column and would make the shared update contract fail. The
repository remains the sole owner of chapter-change timestamp semantics.

Version 2 adds the constraint idempotently for an adopted database:

```sql
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'comic_title_keys_comic_id_fkey'
      AND conrelid = 'comic_title_keys'::regclass
  ) THEN
    ALTER TABLE comic_title_keys
      ADD CONSTRAINT comic_title_keys_comic_id_fkey
      FOREIGN KEY (comic_id) REFERENCES comics(id) ON DELETE CASCADE;
  END IF;
END
$$;
```

The down file drops only
`comic_title_keys_comic_id_fkey`; the baseline down file drops the child table,
the parent table, and `comic_id_seq` in that order.

- [ ] **Step 6: Point sqlc at the migration directories and remove duplicates**

Set:

```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "internal/repo/sql/query/comics_sqlite.sql"
    schema: "internal/repo/migrations/sqlite"
    gen:
      go:
        package: "sqlite"
        out: "internal/repo/sql/sqlite"
  - engine: "postgresql"
    queries: "internal/repo/sql/query/comics_postgre.sql"
    schema: "internal/repo/migrations/postgres"
    gen:
      go:
        package: "postgres"
        out: "internal/repo/sql/postgres"
        sql_package: "pgx/v5"
```

Delete the seven old migration/schema files listed in the file map. The
zero-padded names are required because `sqlc` reads migration files
lexicographically while `golang-migrate` treats versions numerically.

Make the dormant sqlc queries parse against the supported schema: change
SQLite `SoftDeleteComicByID` to assign
`last_update = CAST(strftime('%s', 'now') AS INTEGER)` instead of the
nonexistent `updated_at`, and replace PostgreSQL's `unnest(titles)` search with
`titles ILIKE '%' || $1 || '%'`. Do not reintroduce array columns; the REST
repository's pipe-delimited `TEXT` representation is canonical.

- [ ] **Step 7: Run migration and generation checks**

Run:

```bash
cd go_server
go test ./internal/repo/migrations -count=1
go tool -modfile=tool.go.mod sqlc generate
go test ./internal/repo/sql/... -count=1
git diff --check
```

Expected: migration tests report `ok`; sqlc exits `0`; generated models now
use non-null strings and integer timestamps for SQLite and serialized strings
plus integer timestamps for PostgreSQL; the generated packages compile; the
diff has no whitespace errors. These generated type changes are intentional
and are committed with the source migrations.

- [ ] **Step 8: Commit the canonical source**

```bash
git add go_server/internal/repo/migrations go_server/internal/repo/sql go_server/sqlc.yml
git commit -m "feat(db): unify catalog migration sources"
```

### Task 2: Build a Safe, SQLite-Default Migration CLI

**Files:**

- Create: `go_server/internal/repo/migrations/backup.go`
- Replace: `go_server/cmd/migrate/main.go`
- Create: `go_server/cmd/migrate/main_test.go`
- Modify: `go_server/Makefile`
- Modify: `Makefile`

**Interfaces:**

- Consumes: Task 1's `migrations.Runner` and read-only `ReadState`.
- Produces: consistent verified SQLite backups and `up`, bounded `down`, and
  `version` subcommands for both dialects.

- [ ] **Step 1: Write failing backup and command tests**

Add these cases:

```go
func TestRunSQLiteUpBacksUpAndMigratesLegacyDatabase(t *testing.T)
func TestRunSQLiteUpRequiresOfflineConfirmation(t *testing.T)
func TestRunSQLiteUpRefusesOrphanTitleKeys(t *testing.T)
func TestRunSQLiteUpRefusesKeylessActiveComics(t *testing.T)
func TestRunSQLiteUpRefusesDeletedComicKeys(t *testing.T)
func TestRunSQLiteDownRequiresPositiveSteps(t *testing.T)
func TestRunDownRefusesToRemoveBaselineSchema(t *testing.T)
func TestRunSQLiteDownBacksUpBeforeOneStep(t *testing.T)
func TestRunVersionDoesNotChangeSchema(t *testing.T)
func TestRunPostgresRequiresURL(t *testing.T)
```

The first test must assert all of the following:

```go
require.FileExists(t, result.BackupPath)
require.NoError(t, migrations.VerifySQLite(ctx, result.BackupPath))
require.Equal(t, uint(2), result.State.Version)
requireSQLiteForeignKeyCascade(t, dbPath)
```

The orphan test inserts a key whose `comic_id` does not exist, calls `run`, and
then proves the original table and orphan row are unchanged. The error must
contain `orphan title keys: 1`. The version test uses a legacy file, snapshots
its `sqlite_master` rows, runs `version`, and proves both that the snapshot is
unchanged and that `catalog_schema_migrations` is still absent.

- [ ] **Step 2: Run the command tests and verify they fail**

Run:

```bash
cd go_server
go test ./cmd/migrate -count=1
```

Expected: build failure because `run`, `BackupSQLite`, and `VerifySQLite` do not
exist.

- [ ] **Step 3: Implement verified SQLite backup**

`BackupSQLite` must create the directory, checkpoint WAL, use SQLite's
`VACUUM INTO` for a consistent snapshot, and verify the snapshot before
returning it:

```go
func BackupSQLite(ctx context.Context, dbPath string, backupDir string) (string, error) {
	if strings.TrimSpace(dbPath) == "" {
		return "", errors.New("sqlite path is required")
	}
	if _, err := os.Stat(dbPath); err != nil {
		return "", fmt.Errorf("stat sqlite source: %w", err)
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	db, err := sql.Open("sqlite", SQLiteDSN(dbPath))
	if err != nil {
		return "", err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return "", fmt.Errorf("checkpoint sqlite before backup: %w", err)
	}
	name := fmt.Sprintf(
		"comics_pre_migrate_%s.db",
		time.Now().UTC().Format("20060102_150405.000000000"),
	)
	dest := filepath.Join(backupDir, name)
	quoted := strings.ReplaceAll(dest, "'", "''")
	if _, err := db.ExecContext(ctx, `VACUUM INTO '`+quoted+`'`); err != nil {
		return "", fmt.Errorf("backup sqlite: %w", err)
	}
	if err := VerifySQLite(ctx, dest); err != nil {
		_ = os.Remove(dest)
		return "", fmt.Errorf("verify sqlite backup: %w", err)
	}
	return dest, nil
}
```

`VerifySQLite` opens `SQLiteReadOnlyDSN(dbPath)`, requires
exactly one `PRAGMA integrity_check` row equal to `ok`, and returns all other
rows as an error. It never repairs or writes the backup.

- [ ] **Step 4: Replace the migration CLI**

The command contract is:

```text
migrate up      --driver sqlite|postgres [connection flags]
migrate down    --driver sqlite|postgres --steps N [connection flags]
migrate version --driver sqlite|postgres [connection flags]
```

Connection flags:

```text
--sqlite-path ../db/comics.db
--postgres-url "$COMICS_POSTGRES_URL"
--backup-dir ../db/backups
--backup-ref db/backups/postgres_catalog_20261026_090500.dump
--confirm-offline
```

Defaults come from `COMICS_DB_DRIVER`, `COMICS_SQLITE_PATH`,
`COMICS_POSTGRES_URL`, and `DATABASE_URL`, with SQLite as the default. `down`
must call `Steps(-N)` through `Runner.Down`; it must never call
`migrate.Down()`. SQLite `up` and `down` always create an automatic verified
backup and refuse to run without `--confirm-offline`. PostgreSQL `down` refuses
to run unless `--backup-ref` names an existing non-empty file. Before either
dialect runs down, read the current state and require
`steps < state.Version`; return
`down would remove the catalog baseline; restore a verified backup instead`
rather than allowing version `0`.

For a SQLite mutation, the command order is fixed: parse and validate flags;
for an existing catalog, open `SQLiteReadOnlyDSN` for state and
orphan/keyless preflight; close it; create and verify the backup; only then
open `SQLiteDSN` and call `migrations.Open`.
This matters because the `golang-migrate` driver creates its version table
during `Open`. The `version` subcommand calls `migrations.ReadState` directly
over a read-only connection and never calls `Open`, so inspecting a legacy
database creates no migration table.

`up` also supports a genuinely new path. If the source file is absent, the
command must first create its parent directory, open the exact path through
`SQLiteDSN`, `PingContext`, and close it, then call `BackupSQLite` to create a
verified empty rollback snapshot before migrations run. `BackupSQLite` itself
never creates an implicit source and returns `os.ErrNotExist` through its
wrapped error for a missing path. `down` and `version` require the source file
to exist and never create it.

Use this testable entry point:

```go
type commandResult struct {
	State      migrations.State
	BackupPath string
}

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) (commandResult, error)
```

The command imports both `github.com/jackc/pgx/v5/stdlib` and
`modernc.org/sqlite` for their `database/sql` driver registrations. It opens
mutable SQLite connections with `migrations.SQLiteDSN`, inspection connections
with `migrations.SQLiteReadOnlyDSN`, and PostgreSQL with the supplied URL; only
mutation paths transfer an opened `*sql.DB` to `migrations.Open`.

Before migrating an adopted SQLite database, run:

```sql
SELECT COUNT(*)
FROM comic_title_keys k
LEFT JOIN comics c ON c.id = k.comic_id
WHERE c.id IS NULL;

SELECT COUNT(*)
FROM comics c
WHERE c.deleted = 0
  AND NOT EXISTS (
    SELECT 1
    FROM comic_title_keys k
    WHERE k.comic_id = c.id
  );

SELECT COUNT(*)
FROM comic_title_keys k
JOIN comics c ON c.id = k.comic_id
WHERE c.deleted <> 0;
```

If `comics` has rows but `comic_title_keys` is absent, fail with
`catalog has comics but no title-key table; complete discovery repair first`.
If the second query is nonzero, return:

```go
fmt.Errorf(
	"active comics without discovery keys: %d; complete discovery repair first",
	keylessActive,
)
```

If the third query is nonzero, return
`deleted comics with discovery keys: N; complete discovery repair first`.

After migration, require `PRAGMA integrity_check = ok`,
`PRAGMA foreign_key_check` returns no rows, and
`PRAGMA foreign_keys = 1`.

Successful output is stable:

```text
backup=/tmp/comics-backups/comics_pre_migrate_20261026_090500.000000000.db
dialect=sqlite previous=0 current=2 dirty=false
integrity=ok foreign_keys=on foreign_key_check=ok
```

The timestamped path varies, but the `backup=`, `dialect=`, and verification
keys do not.

- [ ] **Step 5: Fix both Makefile surfaces**

In `go_server/Makefile`:

```make
MIGRATE_DRIVER ?= sqlite
MIGRATE_STEPS ?=
MIGRATE_BACKUP ?=
MIGRATE_BACKUP_DIR ?= ../db/backups
MIGRATE_CONFIRM_OFFLINE ?=

go-migrate-up:
	@test "$(MIGRATE_DRIVER)" != "sqlite" -o "$(MIGRATE_CONFIRM_OFFLINE)" = "yes" || (echo "Set MIGRATE_CONFIRM_OFFLINE=yes after stopping the API" && exit 2)
	go run ./cmd/migrate up \
		--driver "$(MIGRATE_DRIVER)" \
		--sqlite-path "$${COMICS_SQLITE_PATH:-../db/comics.db}" \
		--postgres-url "$${COMICS_POSTGRES_URL:-$${DATABASE_URL}}" \
		--backup-dir "$(MIGRATE_BACKUP_DIR)" \
		$$(test "$(MIGRATE_CONFIRM_OFFLINE)" = "yes" && echo --confirm-offline)

go-migrate-down:
	@test -n "$(MIGRATE_STEPS)" || (echo "Set MIGRATE_STEPS to a positive integer" && exit 2)
	@test "$(MIGRATE_DRIVER)" != "sqlite" -o "$(MIGRATE_CONFIRM_OFFLINE)" = "yes" || (echo "Set MIGRATE_CONFIRM_OFFLINE=yes after stopping the API" && exit 2)
	go run ./cmd/migrate down \
		--driver "$(MIGRATE_DRIVER)" \
		--steps "$(MIGRATE_STEPS)" \
		--sqlite-path "$${COMICS_SQLITE_PATH:-../db/comics.db}" \
		--postgres-url "$${COMICS_POSTGRES_URL:-$${DATABASE_URL}}" \
		--backup-dir "$(MIGRATE_BACKUP_DIR)" \
		--backup-ref "$(MIGRATE_BACKUP)" \
		$$(test "$(MIGRATE_CONFIRM_OFFLINE)" = "yes" && echo --confirm-offline)

go-migration-test:
	go test ./internal/repo/migrations ./cmd/migrate -count=1
```

The root Makefile forwards `MIGRATE_DRIVER`, `MIGRATE_STEPS`,
`MIGRATE_BACKUP`, `MIGRATE_BACKUP_DIR`, `MIGRATE_CONFIRM_OFFLINE`, and an
absolute `COMICS_SQLITE_PATH`. Update `.PHONY` and help text at both levels.

- [ ] **Step 6: Verify no migration can run from the wrong working directory**

Run from the repository root:

```bash
make go-migration-test
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
COMICS_SQLITE_PATH="$tmp/comics.db" MIGRATE_BACKUP_DIR="$tmp/backups" MIGRATE_CONFIRM_OFFLINE=yes make -C go_server go-migrate-up
```

Expected: tests pass; the temporary database reaches version `2`; no
`internal/db/migrations: file does not exist` error appears.

- [ ] **Step 7: Commit the CLI and backup boundary**

```bash
git add Makefile go_server/Makefile go_server/cmd/migrate go_server/internal/repo/migrations/backup.go
git commit -m "feat(db): add safe catalog migration CLI"
```

### Task 3: Enforce Schema Currency and SQLite Foreign Keys at Runtime

**Files:**

- Modify: `go_server/internal/repo/comics/sql_repository.go`
- Modify: `go_server/internal/repo/comics/title_keys.go`
- Modify: `go_server/internal/repo/comics/sqlite_health.go`
- Modify: `go_server/internal/repo/comics/sql_repository_test.go`
- Modify: `go_server/internal/repo/comics/discovery_audit_test.go`
- Modify: `go_server/cmd/db_maint/main.go`
- Modify: `go_server/cmd/db_maint/main_test.go`
- Modify: `go_server/Makefile`
- Modify: `Makefile`
- Modify: `go_server/api/route/comics_test.go`
- Modify: `go_server/internal/scrape/scrape_test.go`
- Modify: `go_server/internal/repo/comics_repo.go`

**Interfaces:**

- Consumes: `migrations.ValidateCurrent`.
- Produces: constructors that validate rather than mutate schema, and SQLite
  connections with per-connection foreign keys enabled.

- [ ] **Step 1: Add failing runtime schema/FK tests**

Add:

```go
func TestNewSQLiteComicRepositoryRejectsUnmigratedSchema(t *testing.T)
func TestSQLiteRepositoryEnablesForeignKeys(t *testing.T)
func TestSQLiteRepositoryRejectsOrphanDiscoveryKey(t *testing.T)
func TestSQLiteRawComicDeleteCascadesDiscoveryKeys(t *testing.T)
func TestSQLiteHealthRejectsForeignKeyViolations(t *testing.T)
func TestSQLiteReadOnlyAuditOpensLegacySchemaWithoutWriting(t *testing.T)
```

The first test expects `errors.Is(err, migrations.ErrSchemaNotCurrent)`.
The second reads `PRAGMA foreign_keys` through `repo.db` and expects `1`.
The cascade test must execute raw `DELETE FROM comics WHERE id = ?` rather than
`repo.Delete`, then assert zero key rows; this prevents a manual key delete from
masquerading as schema-level cascade behavior.

For the health-violation fixture, migrate and close the normal connection, open
a one-off SQLite connection without the foreign-key DSN pragma, set
`PRAGMA foreign_keys=OFF`, insert an orphan key, and close it. Reopening through
`NewSQLiteComicRepository` must fail with the table, row ID, parent, and FK ID
reported by `foreign_key_check`.

The read-only test opens a version-0 legacy fixture with
`NewSQLiteComicRepositoryReadOnly`, runs `AuditDiscoveryIntegrity`, and compares
the DB/WAL/SHM digest set before and after. This constructor must remain usable
for discovery audit/export before the offline migration.

- [ ] **Step 2: Run the focused tests and verify current behavior fails**

Run:

```bash
cd go_server
go test ./internal/repo/comics -run 'Test(NewSQLite|SQLiteRepository|SQLiteHealth)' -count=1
```

Expected: the unmigrated schema is currently accepted, `PRAGMA foreign_keys`
returns `0`, or the orphan insert succeeds.

- [ ] **Step 3: Enable foreign keys in every SQLite connection**

Remove `sqliteBusyTimeoutMS` and the repository-local DSN construction. Delegate
to the migration package so runtime, migration, backup, and tests share exactly
one pragma policy:

```go
func sqliteDSN(path string) string {
	return migrations.SQLiteDSN(path)
}
```

Keep `SetMaxOpenConns(1)` and `SetMaxIdleConns(1)`. Immediately after opening,
query `PRAGMA foreign_keys`; return
`sqlite foreign key enforcement is disabled` unless it is `1`.

Extend health validation with:

```go
func sqliteForeignKeysOK(ctx context.Context, db *sql.DB) (bool, string) {
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return false, err.Error()
	}
	defer rows.Close()
	if rows.Next() {
		var table string
		var rowID sql.NullInt64
		var parent string
		var fkID int
		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			return false, err.Error()
		}
		return false, fmt.Sprintf("table=%s rowid=%v parent=%s fk=%d", table, rowID, parent, fkID)
	}
	return rows.Err() == nil, "ok"
}
```

Foreign-key violations are not repaired with `REINDEX` or `VACUUM`; startup
must fail and point to restore/repair. Call `sqliteForeignKeysOK` after a clean
integrity result and again after any successful integrity repair; never enter
the repair loop because of a foreign-key violation.

- [ ] **Step 4: Remove runtime DDL and validate current version**

Delete `postgresComicSchema`, `sqlitePerformanceIndexes`,
`ensureSQLitePerformanceIndexes`, `EnsurePostgresComicSchema`,
`comicTitleKeysDDL`, and every startup call to
`ensureComicTitleKeysSchema`/backfill. Constructors follow this order:

```go
// SQLite
open -> ping -> configure pragmas/health -> migrations.ValidateCurrent -> return

// PostgreSQL
open -> ping -> migrations.ValidateCurrent -> return
```

No constructor executes `CREATE`, `ALTER`, index creation, backfill, or schema
migration. `NewSQLiteComicRepositoryReadOnly` is the deliberate exception to
the version gate: it validates required table/column presence without writing
or requiring `catalog_schema_migrations`, and opens
`migrations.SQLiteReadOnlyDSN`. `cmd/db_maint` also opens the offline database
directly so the discovery repair can run before version 2 exists.

In the repository's private hard-delete implementation, remove the explicit
`DeleteDiscoveryKeysForComic` call and delete only the parent row. The public
`Delete` transaction wrapper from the discovery plan remains; both dialects
now exercise the schema's `ON DELETE CASCADE` rather than maintaining a second
application-level cascade.

Remove `DropLegacyIdentityKeyColumn`, `dropLegacyIdentityKeyColumn`,
`sqliteTableHasColumn`, and the `--drop-legacy-identity-key` branch from
`cmd/db_maint`. Retain the discovery plan's guarded
`--apply-discovery-plan` data-repair path. This prevents `db_maint` from
remaining a second schema migration mechanism; workstream 8 owns any remaining
legacy-column cleanup decision. Update both Make help descriptions so
`go-db-maint` describes reviewed discovery repair and no longer gives the
removed identity-column example.

- [ ] **Step 5: Migrate every test fixture instead of copying schema strings**

Replace hand-written target schemas in repository, route, and scrape tests with
a helper that:

```go
db, err := sql.Open("sqlite", migrations.SQLiteDSN(dbPath))
require.NoError(t, err)
runner, err := migrations.Open(db, migrations.DialectSQLite)
require.NoError(t, err)
_, err = runner.Up(context.Background())
require.NoError(t, err)
require.NoError(t, runner.Close())
repo, err := NewSQLiteComicRepository(dbPath)
```

The importer source fixture remains a source-format fixture because it verifies
reading an existing SQLite catalog; target fixtures use canonical migrations.

- [ ] **Step 6: Retire the legacy gRPC startup migrator**

In `internal/repo/comics_repo.go`, remove `migrationSource`, all
`golang-migrate` imports, `runMigrations`, and its call from `NewComicsRepo`.
The gRPC repository may connect only to an already migrated PostgreSQL
catalog. This removes a second executable migration path without claiming that
the divergent gRPC data model is supported.

- [ ] **Step 7: Verify runtime behavior**

Run:

```bash
cd go_server
go test ./internal/repo/comics ./api/route ./internal/scrape -count=1
go test ./... -count=1
```

Expected: both commands pass; no ordinary repository/route/scrape success
fixture contains `CREATE TABLE comics` (explicit legacy-rejection fixtures and
the importer source-format fixture are the only exceptions); opening an
unmigrated target reports the migration command instead of creating tables.

- [ ] **Step 8: Commit runtime schema enforcement**

```bash
git add Makefile go_server/Makefile go_server/internal/repo/comics go_server/internal/repo/comics_repo.go go_server/api/route/comics_test.go go_server/internal/scrape/scrape_test.go go_server/cmd/db_maint
git commit -m "feat(db): enforce migrated catalog schemas"
```

### Task 4: Lock the Discovery Plan's Atomic Write Boundary to the Final Schema

**Files:**

- Modify: `go_server/domain/comic.go`
- Modify: `go_server/internal/repo/comics/sql_repository_test.go`
- Modify: `go_server/internal/repo/comics/sql_repository_identity_test.go`
- Modify: `go_server/internal/usecase/comics_test.go`
- Modify: `go_server/api/route/comics_test.go`

**Interfaces:**

- Consumes: discovery-plan Tasks 2 and 5, including strict
  `SyncDiscoveryKeys`, normalized repository writes, `Resolver.Resolve`, and
  transactional create/update use cases.
- Produces: migration-backed regression tests and explicit transactional
  comments on the unchanged `ComicRepository` interface, plus REST-to-database
  rollback coverage.

- [ ] **Step 1: Rebase onto and verify the completed discovery plan**

Before editing shared files:

```bash
git rebase main
cd go_server
go test ./internal/identity ./internal/discovery ./internal/repo/comics ./internal/usecase ./cmd/import_comics_postgres -count=1
```

Expected: all packages pass; strict conflicts satisfy both
`errors.Is(err, domain.ErrDiscoveryKeyConflict)` and
`errors.Is(err, domain.ErrDuplicateComic)`.

Retain the dependency's normalization, deleted-key, ambiguity, import
preflight, and transaction wrappers. This task verifies those wrappers against
the final migrated schema instead of adding a second implementation.

- [ ] **Step 2: Add service transaction-boundary tests**

Wrap `memoryComicRepo` with a recorder that passes itself into `WithTx` and
counts reads and writes made at depth zero. Add:

```go
func TestComicServiceCreateResolvesAndWritesInsideOneTransaction(t *testing.T)
func TestComicServiceUpdateReadsChecksAndWritesInsideOneTransaction(t *testing.T)
```

The exact assertions are:

```go
require.Equal(t, 1, repo.transactions)
require.Zero(t, repo.readsOutsideTx)
require.Zero(t, repo.writesOutsideTx)
```

Run these tests against the pre-discovery baseline once to record the intended
failure: create/update access the repository outside `WithTx`. After rebasing
the dependency, they must pass without another use-case rewrite.

- [ ] **Step 3: Add migration-backed SQL rollback tests**

Use the strict conflict contract and the Task 1 schema:

```go
func TestSQLComicRepositoryCreateRollsBackRowWhenKeySyncFails(t *testing.T) {
	repo := newTestSQLiteRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, domain.Comic{Titles: []string{"Atomic hero"}, ComType: 3})
	require.NoError(t, err)

	_, err = repo.Create(ctx, domain.Comic{Titles: []string{"atomic hero"}, ComType: 3})
	require.ErrorIs(t, err, domain.ErrDiscoveryKeyConflict)
	requireComicCount(t, repo.db, 1)
	requireKeylessActiveCount(t, repo.db, 0)
}

func TestSQLComicRepositoryUpdateRollsBackRowAndKeysOnConflict(t *testing.T) {
	repo := newTestSQLiteRepo(t)
	ctx := context.Background()
	first := mustCreateComic(t, repo, "First", 3)
	second := mustCreateComic(t, repo, "Second", 3)
	second.Titles = []string{"First"}
	second.CurrentChap = 99

	_, err := repo.Update(ctx, second)
	require.ErrorIs(t, err, domain.ErrDiscoveryKeyConflict)
	stored, err := repo.Get(ctx, second.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"Second"}, stored.Titles)
	require.NotEqual(t, 99, stored.CurrentChap)
	requireKeysForComic(t, repo.db, second.ID, "second")
	requireKeysForComic(t, repo.db, first.ID, "first")
}
```

- [ ] **Step 4: Prove the authenticated REST paths roll back end to end**

Using Task 3's migration-backed SQLite route fixture and workstream 1's admin
request helper, add:

```go
func TestRESTCreateConflictRollsBackComicAndKeys(t *testing.T)
func TestRESTUpdateConflictRollsBackComicAndKeys(t *testing.T)
```

Seed an owner for both tests and a second candidate for update. POST the
owner's normalized discovery title and assert HTTP `409`, response code
`discovery_key_conflict`, the original comic count, and
`keyless_active=0`. PUT the candidate with the owner's title and
`current_chap=99`; assert the same `409`, then read through the repository and
prove the candidate's original title, chapter, and keys remain. These tests
exercise route binding, `ComicService.WithTx`, resolver checks, repository row
write, and key synchronization as one observable boundary.

- [ ] **Step 5: Document the final interface semantics**

Add comments to `domain.ComicRepository`:

```go
// Create atomically persists the normalized comic and its discovery keys.
Create(ctx context.Context, comic Comic) (Comic, error)
// Update atomically persists the normalized comic and replaces its discovery keys.
Update(ctx context.Context, comic Comic) (Comic, error)
// WithTx commits fn as one transaction; nested calls reuse the active transaction.
WithTx(ctx context.Context, fn func(ComicRepository) error) error
```

The required final call chain is:

```text
REST Create -> ComicService.WithTx -> Resolver.Resolve -> SQL Create -> SyncDiscoveryKeys -> commit
REST Update -> ComicService.WithTx -> Get -> Resolver.Resolve(exclude self) -> SQL Update -> SyncDiscoveryKeys -> commit
```

No inner call may open and commit an independent transaction when `r.tx != nil`.

- [ ] **Step 6: Run unit, route, and scrape regression tests**

Run:

```bash
cd go_server
go test ./internal/usecase ./internal/repo/comics ./api/route ./internal/scrape -count=1
```

Expected: all tests pass; the rollback tests leave `keyless_active=0`; merge and
scrape continue to reuse the outer transaction.

- [ ] **Step 7: Commit the final-schema atomicity regression**

```bash
git add go_server/domain/comic.go go_server/internal/repo/comics/sql_repository_test.go go_server/internal/repo/comics/sql_repository_identity_test.go go_server/internal/usecase/comics_test.go go_server/api/route/comics_test.go
git commit -m "test(db): lock catalog transaction boundaries"
```

### Task 5: Run One Repository Contract Against SQLite and PostgreSQL

**Files:**

- Create: `go_server/internal/repo/comics/repository_contract_test.go`
- Create: `go_server/internal/repo/comics/repository_contract_postgres_test.go`
- Delete: `go_server/internal/repo/comics/sql_repository_postgres_test.go`
- Modify: `go_server/go.mod`
- Modify: `go_server/go.sum`
- Modify: `go_server/Makefile`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**

- Consumes: Task 1 migrations, Task 4 atomic repository semantics, and the
  discovery plan's `runDiscoveryContract(t, repo)`.
- Produces: `runRepositoryContract(t, repo)` shared unchanged by both dialect
  harnesses; the PostgreSQL container runs both contracts.

- [ ] **Step 1: Add the shared failing contract**

Create one function with these named subtests:

```go
func runRepositoryContract(t *testing.T, repo *SQLComicRepository) {
	t.Helper()
	t.Run("create stores row and keys", contractCreateStoresRowAndKeys(repo))
	t.Run("conflicting create rolls back row", contractCreateConflictRollsBack(repo))
	t.Run("conflicting update rolls back row and keys", contractUpdateConflictRollsBack(repo))
	t.Run("callback error rolls back transaction", contractCallbackErrorRollsBack(repo))
	t.Run("delete cascades keys", contractDeleteCascadesKeys(repo))
	t.Run("not found is typed", contractNotFound(repo))
	t.Run("list filters and sort agree", contractListFiltersAndSort(repo))
}
```

Each subtest clears `comic_title_keys` and `comics` first. The cascade case
performs a raw parent delete and verifies the child count. The conflict cases
assert `errors.Is(err, domain.ErrDiscoveryKeyConflict)`, unchanged row count,
unchanged comic values, and no keyless active records.

Add `TestSQLiteRepositoryContract`, which migrates a temporary file, opens
`NewSQLiteComicRepository`, and calls the shared function.

- [ ] **Step 2: Run the SQLite contract and verify the current implementation fails**

Run:

```bash
cd go_server
go test ./internal/repo/comics -run '^TestSQLiteRepositoryContract$' -count=1
```

Expected on the pre-discovery baseline: at least the conflicting-create or
conflicting-update rollback case fails. After the dependency and Task 4,
SQLite passes.

- [ ] **Step 3: Add a disposable PostgreSQL harness**

Add the module at the already-used testcontainers version:

```bash
cd go_server
go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.36.0
```

The integration-tagged harness uses:

```go
container, err := postgres.Run(
	ctx,
	"postgres:16-alpine",
	postgres.WithDatabase("comics_contract"),
	postgres.WithUsername("comics"),
	postgres.WithPassword("comics"),
	postgres.BasicWaitStrategies(),
)
require.NoError(t, err)
testcontainers.CleanupContainer(t, container)

url, err := container.ConnectionString(ctx, "sslmode=disable")
require.NoError(t, err)
migratePostgresForTest(t, url)
repo, err := NewPostgresComicRepository(url)
require.NoError(t, err)
t.Cleanup(func() { require.NoError(t, repo.Close()) })
runRepositoryContract(t, repo)
resetPostgresContractTables(t, repo.db)
runDiscoveryContract(t, repo)
```

Define the migration helper in the same test file:

```go
func migratePostgresForTest(t *testing.T, postgresURL string) {
	t.Helper()
	db, err := sql.Open("pgx", postgresURL)
	require.NoError(t, err)
	runner, err := migrations.Open(db, migrations.DialectPostgres)
	require.NoError(t, err)
	_, err = runner.Up(context.Background())
	require.NoError(t, err)
	require.NoError(t, runner.Close())
}

func resetPostgresContractTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`TRUNCATE TABLE comic_title_keys, comics RESTART IDENTITY`)
	require.NoError(t, err)
}
```

Delete the environment-skipped `sql_repository_postgres_test.go`; the
testcontainer harness now executes both repository and discovery contracts, so
CI cannot silently skip PostgreSQL parity.

- [ ] **Step 4: Add contract targets and CI**

`go_server/Makefile`:

```make
repo-contract-test:
	go test ./internal/repo/comics -run '^TestSQLite(Repository|Discovery)Contract$$' -count=1
	go test -tags=integration ./internal/repo/comics -run '^TestPostgresRepositoryContract$$' -count=1
```

Add `repo-contract-test` to both `check-go` dependency lists and forward it
from the root Makefile. In `.github/workflows/ci.yml`, replace the bare Go test
step with:

```yaml
- run: make go-test
- run: make repo-contract-test
```

GitHub-hosted Ubuntu runners provide Docker for the PostgreSQL container.

- [ ] **Step 5: Run both dialects**

Run:

```bash
cd go_server
make repo-contract-test
```

Expected:

```text
ok  	comics/internal/repo/comics
ok  	comics/internal/repo/comics
```

No test prints `COMICS_POSTGRES_URL not set` or `SKIP`.

- [ ] **Step 6: Commit the parity gate**

```bash
git add .github/workflows/ci.yml Makefile go_server/Makefile go_server/go.mod go_server/go.sum go_server/internal/repo/comics
git commit -m "test(db): enforce SQLite and Postgres repository parity"
```

### Task 6: Wire Canonical Migrations into the Atomic PostgreSQL Import

**Files:**

- Modify: `go_server/cmd/import_comics_postgres/main.go`
- Modify: `go_server/cmd/import_comics_postgres/main_test.go`
- Create: `go_server/cmd/import_comics_postgres/integration_test.go`
- Modify: `go_server/Makefile`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**

- Consumes: the discovery plan's
  `prepareImportComics([]sourceComic) ([]preparedComic, error)` and
  one-transaction `replaceTargetComics`, plus Task 1 migrations.
- Produces:
  `replaceTargetComics(context.Context, *sql.DB, []preparedComic) (verifyCounts, importBackup, error)`
  and an integration gate that proves post-mutation failures restore the
  original target.

- [ ] **Step 1: Write failing PostgreSQL import integration tests**

Use the same `postgres:16-alpine` testcontainer pattern as Task 5. Add:

```go
func TestImportMigratesEmptyPostgresTarget(t *testing.T)
func TestImportRollbackPreservesOriginalTargetAfterInsertFailure(t *testing.T)
func TestImportRollbackRestoresSequenceAfterVerificationFailure(t *testing.T)
func TestImportSuccessCommitsRowsKeysSequenceAndBackups(t *testing.T)
```

For the rollback test, seed an original target row and key, then prepare two
source rows with the same `ID` and different non-conflicting titles. In-memory
key preflight succeeds; the second comic insert fails on the primary key after
the target delete and first insert have already executed:

```go
prepared, err := prepareImportComics([]sourceComic{
	{ID: 100, Titles: "First imported title", ComType: 3},
	{ID: 100, Titles: "Second imported title", ComType: 3},
})
require.NoError(t, err)

_, _, err = replaceTargetComics(context.Background(), target, prepared)
require.Error(t, err)
requireTargetComic(t, target, originalID, "Original target")
requireTargetKey(t, target, originalID, "series", "originaltarget")
requireNoImportBackupTables(t, target)
```

The absence of backup tables is expected on failure because their creation is
inside the rolled-back transaction.

For the sequence rollback test, first commit
`ALTER SEQUENCE comic_id_seq RESTART WITH 42`, then call
`replaceTargetComics` with a package-local handcrafted active `preparedComic`
whose `Keys` is empty. The replacement reaches sequence adjustment and then
fails `keyless_active` verification. Assert the original rows and backup-table
absence as above, and assert `last_value`/`is_called` still describe restart
value `42`.

- [ ] **Step 2: Run the importer integration tests and verify the migration gap**

Run:

```bash
cd go_server
go test -tags=integration ./cmd/import_comics_postgres -run '^TestImport' -count=1
```

Expected after Task 3 but before this task: FAIL because
`EnsurePostgresComicSchema` has been removed and an empty target has no
canonical migrated schema.

- [ ] **Step 3: Add the canonical migration call**

Before opening the target for replacement, migrate it with the Task 1 runner:

```go
func migratePostgresTarget(ctx context.Context, postgresURL string) error {
	db, err := sql.Open("pgx", postgresURL)
	if err != nil {
		return err
	}
	runner, err := migrations.Open(db, migrations.DialectPostgres)
	if err != nil {
		return err
	}
	_, migrateErr := runner.Up(ctx)
	closeErr := runner.Close()
	return errors.Join(migrateErr, closeErr)
}
```

Call `migratePostgresTarget` once, then open the normal target connection.
Remove `EnsurePostgresComicSchema`; do not add inline DDL or another migration
directory.

- [ ] **Step 4: Preserve the dependency's single transaction and expose its result**

Keep the discovery plan's preflight and this transaction order:

1. Create timestamped tables such as
   `comics_backup_20261026_090500_000000000` and
   `comic_title_keys_backup_20261026_090500_000000000` when the target is
   non-empty.
2. `TRUNCATE TABLE comic_title_keys, comics RESTART IDENTITY`.
3. Insert every normalized prepared comic.
4. Insert every prepared title key.
5. Transactionally restart `comic_id_seq`.
6. Query `verifyCounts` before commit.
7. Require total parity, `KeylessActive == 0`, `DeletedKeyRows == 0`, and
   `DiscoveryOwners == Active`.
8. Commit only after all checks pass.

Return the committed counts and backup names:

```go
type importBackup struct {
	ComicsTable   string
	TitleKeysTable string
}

func replaceTargetComics(
	ctx context.Context,
	db *sql.DB,
	comics []preparedComic,
) (verifyCounts, importBackup, error)
```

Do not use `setval`: PostgreSQL sequence function effects are not rolled back.
Compute `nextID` as one greater than the maximum prepared ID, or `1` for an
empty import, and execute the fixed-identifier statement inside the replacement
transaction:

```go
_, err := tx.ExecContext(
	ctx,
	fmt.Sprintf(`ALTER SEQUENCE comic_id_seq RESTART WITH %d`, nextID),
)
```

`nextID` is a checked positive integer derived from prepared rows, never raw
user text. `ALTER SEQUENCE ... RESTART` is transactional, so the post-sequence
verification-failure test can prove rollback of rows, keys, backup-table DDL,
and sequence state.

Backup identifiers use only fixed prefixes plus
`time.Now().UTC().Format("20060102_150405_000000000")`. Never interpolate a
user-provided identifier.

- [ ] **Step 5: Print success only after commit**

For the two-row success fixture, assert:

```text
Imported 2 comics into Postgres
Verified active=2 tracked=1 cover_invisible=1 discovery_owners=2 keyless_active=0 deleted_key_rows=0
Backup tables: comics_backup_20261026_090500_000000000, comic_title_keys_backup_20261026_090500_000000000
```

Production counts and UTC suffixes reflect the imported source. On any
migration, preparation, insert, key, sequence, verification, or commit error,
print no success line.

- [ ] **Step 6: Add importer integration targets**

Add:

```make
import-contract-test:
	go test -tags=integration ./cmd/import_comics_postgres -run '^TestImport' -count=1
```

Add `import-contract-test` to both `check-go` dependency lists and forward it
from the root Makefile. Add `make import-contract-test` to the Go CI job after
the repository contract step so the integration-tagged rollback test cannot be
silently omitted.

- [ ] **Step 7: Run importer and repository contracts**

Run:

```bash
cd go_server
go test ./cmd/import_comics_postgres -count=1
make import-contract-test
make repo-contract-test
```

Expected: all commands pass; the rollback integration test proves the original
target row and key survive a failure after replacement begins.

- [ ] **Step 8: Commit canonical atomic import**

```bash
git add .github/workflows/ci.yml Makefile go_server/Makefile go_server/cmd/import_comics_postgres
git commit -m "fix(db): migrate and verify Postgres imports atomically"
```

### Task 7: Rehearse, Roll Back Safely, and Document the Cutover

**Files:**

- Modify: `Makefile`
- Modify: `go_server/Makefile`
- Modify: `go_server/README.md`
- Modify: `docs/runbooks/discovery-data-repair.md`

**Interfaces:**

- Consumes: completed discovery repair, migration CLI, backup verification, and
  repository contracts.
- Produces: a non-destructive migration rehearsal and an operator runbook.

- [ ] **Step 1: Strengthen root backup and restore targets**

After `db-backup` creates its file, run `PRAGMA integrity_check` on the backup
and delete it if verification is not exactly `ok`.

`db-restore` must require `CONFIRM_OFFLINE=yes`, verify the selected backup
before touching the live path, copy to `$(DB_FILE).restore`, atomically rename
that file over `$(DB_FILE)`, and remove stale `-wal`/`-shm` files only after the
offline confirmation. Remove the sidecars before the final rename so they
cannot be replayed against the restored main file:

```make
db-restore:
	@test "$(CONFIRM_OFFLINE)" = "yes" || (echo "Set CONFIRM_OFFLINE=yes after stopping the API" && exit 2)
	@test -n "$(BACKUP)" || (echo "Set BACKUP=db/backups/comics_backup_20261026_090500.db" && exit 2)
	@test -f "$(BACKUP)" || (echo "Backup not found: $(BACKUP)" && exit 1)
	@result=$$(sqlite3 -readonly "$(BACKUP)" "PRAGMA integrity_check;"); \
		test "$$result" = "ok" || (echo "Backup integrity failed: $$result" && exit 1)
	@trap 'rm -f "$(DB_FILE).restore"' EXIT; \
		cp "$(BACKUP)" "$(DB_FILE).restore"; \
		result=$$(sqlite3 -readonly "$(DB_FILE).restore" "PRAGMA integrity_check;"); \
		test "$$result" = "ok" || (echo "Restore copy integrity failed: $$result" && exit 1); \
		rm -f "$(DB_FILE)-wal" "$(DB_FILE)-shm"; \
		mv "$(DB_FILE).restore" "$(DB_FILE)"
	@echo "Restored $(DB_FILE) from $(BACKUP); integrity=ok"
```

The help text must say that the API and scraper must be stopped first.

- [ ] **Step 2: Add a copy-only rehearsal target**

Add `db-migrate-rehearse` at the root. It creates a temporary directory, copies
`$(DB_FILE)` with SQLite's consistent backup API, runs migration against only
that copy, verifies integrity/FKs, prints the source and copy hashes, and
removes the directory via `trap`. Require offline confirmation so the source
hash is a deterministic no-write assertion rather than racing the scheduler.

Core recipe:

```make
db-migrate-rehearse:
	@test "$(CONFIRM_OFFLINE)" = "yes" || (echo "Set CONFIRM_OFFLINE=yes after stopping the API" && exit 2)
	@tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	before=$$(shasum -a 256 "$(DB_FILE)" | awk '{print $$1}'); \
	sqlite3 "$(DB_FILE)" ".backup '$$tmp/comics.db'"; \
	test "$$(sqlite3 -readonly "$$tmp/comics.db" "PRAGMA integrity_check;")" = "ok"; \
	COMICS_SQLITE_PATH="$$tmp/comics.db" \
		MIGRATE_BACKUP_DIR="$$tmp/backups" \
		MIGRATE_CONFIRM_OFFLINE=yes \
		$(MAKE) -C go_server go-migrate-up; \
	sqlite3 -readonly "$$tmp/comics.db" \
		"PRAGMA foreign_keys=ON; PRAGMA integrity_check; PRAGMA foreign_keys; PRAGMA foreign_key_check;"; \
	after=$$(shasum -a 256 "$(DB_FILE)" | awk '{print $$1}'); \
	migrated=$$(shasum -a 256 "$$tmp/comics.db" | awk '{print $$1}'); \
	test "$$before" = "$$after"; \
	echo "rehearsal=ok source_unchanged=yes source_sha256=$$after migrated_sha256=$$migrated"
```

The migration command's automatic backup is written under `$$tmp/backups`; the
rehearsal leaves no repository files.

- [ ] **Step 3: Document the exact SQLite production sequence**

Add this runbook to `go_server/README.md`:

```bash
# 1. Verify code and both repository dialects.
make check-go

# 2. Prove discovery repair is complete.
make db-audit-discovery DB_FILE=db/comics.db

# 3. Stop the only SQLite API/scheduler instance.
make dev-down

# 4. Rehearse against a consistent copy; the offline source hash must stay unchanged.
make db-migrate-rehearse CONFIRM_OFFLINE=yes

# 5. Create and verify a named rollback point.
make db-backup

# 6. Apply the offline migration.
make go-migrate-up MIGRATE_DRIVER=sqlite MIGRATE_CONFIRM_OFFLINE=yes

# 7. Verify schema and data before restarting.
make db-check
sqlite3 -readonly db/comics.db \
  "PRAGMA foreign_key_check; SELECT COUNT(*) FROM comics c WHERE c.deleted=0 AND NOT EXISTS (SELECT 1 FROM comic_title_keys k WHERE k.comic_id=c.id);"

# 8. Start one API instance and run the contract/smoke checks.
make dev-up
make repo-contract-test
```

Expected step 7 output is `0` for the keyless query and no rows before it from
`foreign_key_check`.

Document rollback as restore, not reverse migration:

```bash
make dev-down
BACKUP="$(ls -1t db/backups/comics_pre_migrate_*.db | awk 'NR==1 { print; exit }')"
make db-restore BACKUP="$BACKUP" CONFIRM_OFFLINE=yes
make db-check
make dev-up
```

Update `docs/runbooks/discovery-data-repair.md` so every post-Task-7
`db-restore` invocation also passes `CONFIRM_OFFLINE=yes`.

Reverse migration remains available for development/rehearsal:

```bash
make go-migrate-down MIGRATE_DRIVER=sqlite MIGRATE_STEPS=1 MIGRATE_CONFIRM_OFFLINE=yes
```

Production rollback uses the verified file because it restores schema and data
to the same point.

- [ ] **Step 4: Document PostgreSQL optional-path safety**

Document:

```bash
pg_dump "$COMICS_POSTGRES_URL" \
  --format=custom \
  --file="db/backups/postgres_catalog_$(date -u +%Y%m%d_%H%M%S).dump"
make go-migrate-up MIGRATE_DRIVER=postgres
make go-import-postgres
make repo-contract-test
```

PostgreSQL remains optional. These commands do not change
`COMICS_DB_DRIVER=sqlite`, the default Compose profile, or the one-replica
SQLite deployment.

- [ ] **Step 5: Run the rehearsal and full release checks**

Run:

```bash
make db-migrate-rehearse CONFIRM_OFFLINE=yes
make check-go
git status --short
```

Expected:

```text
rehearsal=ok source_unchanged=yes
check-go passes
```

`db/comics.db`, `db/comics.db-wal`, and `db/comics.db-shm` must not appear as
modified. Only intended implementation/documentation files may appear.

- [ ] **Step 6: Commit the runbook and safety targets**

```bash
git add Makefile go_server/Makefile go_server/README.md docs/runbooks/discovery-data-repair.md
git commit -m "docs(db): add catalog migration rollback runbook"
```

## Final Acceptance Checks

Run from the repository root:

```bash
make go-migration-test
make repo-contract-test
make import-contract-test
make go-test
make go-vet
make go-build
make -C go_server sqlc
git diff --exit-code -- go_server/internal/repo/sql/sqlite go_server/internal/repo/sql/postgres
make db-migrate-rehearse CONFIRM_OFFLINE=yes
```

Expected:

- Migration tests pass from empty and legacy SQLite schemas.
- Both SQLite and PostgreSQL execute the same repository contract with no skip.
- A create/update key conflict leaves no new or partially updated comic row.
- Raw parent deletion cascades title keys in both dialects.
- `PRAGMA foreign_keys` is `1` on the runtime SQLite connection.
- A mid-import SQL failure rolls back comics, keys, sequence state, and backup-table DDL.
- API constructors reject an unmigrated catalog and never execute schema DDL.
- `sqlc generate` consumes only the canonical migration directories.
- Rehearsal reports `source_unchanged=yes`.
- SQLite remains the default; PostgreSQL requires explicit configuration.

## Implementation Order and Commit Boundaries

Execute and review in this order:

1. Land/rebase the discovery-data-integrity dependency.
2. `feat(db): unify catalog migration sources`
3. `feat(db): add safe catalog migration CLI`
4. `feat(db): enforce migrated catalog schemas`
5. `test(db): lock catalog transaction boundaries`
6. `test(db): enforce SQLite and Postgres repository parity`
7. `fix(db): migrate and verify Postgres imports atomically`
8. `docs(db): add catalog migration rollback runbook`

Never squash the migration-source, transaction-regression, importer, and
operational commits together: each is an independently testable rollback
boundary. Do not apply the live SQLite migration until the discovery audit is
clean and an approved, verified backup exists.
