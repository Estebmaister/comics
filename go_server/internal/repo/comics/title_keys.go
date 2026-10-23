package comics

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"comics/domain"
	"comics/internal/identity"
)

const comicTitleKeysDDL = `
CREATE TABLE IF NOT EXISTS comic_title_keys (
	comic_id INTEGER NOT NULL,
	scope TEXT NOT NULL,
	title_key TEXT NOT NULL,
	PRIMARY KEY (scope, title_key)
);
CREATE INDEX IF NOT EXISTS idx_comic_title_keys_comic_id ON comic_title_keys(comic_id);
`

func ensureComicTitleKeysSchema(db *sql.DB) error {
	if _, err := db.Exec(comicTitleKeysDDL); err != nil {
		return err
	}
	dialect := dialectSQLite
	if isPostgresDB(db) {
		dialect = dialectPostgres
	}
	return backfillComicTitleKeysIfEmpty(db, dialect)
}

func isPostgresDB(db *sql.DB) bool {
	var version string
	if err := db.QueryRow(`SELECT version()`).Scan(&version); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(version), "postgresql")
}

func backfillComicTitleKeysIfEmpty(db *sql.DB, dialect string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comic_title_keys`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	start := time.Now()
	inserted, err := fastBackfillComicTitleKeys(db, dialect)
	if err != nil {
		return err
	}
	slog.Info("discovery keys backfill complete",
		"rows", inserted,
		"elapsed", time.Since(start).Round(time.Millisecond),
	)
	return nil
}

func fastBackfillComicTitleKeys(db *sql.DB, dialect string) (int, error) {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	repo := &SQLComicRepository{db: db, tx: tx, dialect: dialect}

	insertSQL := `INSERT INTO comic_title_keys (comic_id, scope, title_key) VALUES (` +
		repo.placeholder(1) + `, ` + repo.placeholder(2) + `, ` + repo.placeholder(3) + `)`
	if dialect == dialectSQLite {
		insertSQL = `INSERT OR IGNORE INTO comic_title_keys (comic_id, scope, title_key) VALUES (` +
			repo.placeholder(1) + `, ` + repo.placeholder(2) + `, ` + repo.placeholder(3) + `)`
	} else {
		insertSQL += ` ON CONFLICT (scope, title_key) DO NOTHING`
	}
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	rows, err := tx.QueryContext(ctx, `SELECT id, titles, com_type, deleted FROM comics`)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer rows.Close()

	inserted := 0
	seen := map[string]struct{}{}
	for rows.Next() {
		var id int
		var titles string
		var comType int
		var deleted bool
		if err := rows.Scan(&id, &titles, &comType, &deleted); err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if deleted {
			continue
		}
		for _, title := range splitTitles(titles) {
			scope := discoveryScopeForTitle(title, comType)
			titleKey := discoveryTitleKey(title)
			if titleKey == "" {
				continue
			}
			scoped := scope + "\x00" + titleKey
			if _, ok := seen[scoped]; ok {
				continue
			}
			seen[scoped] = struct{}{}
			result, err := stmt.ExecContext(ctx, id, scope, titleKey)
			if err != nil {
				_ = tx.Rollback()
				return 0, fmt.Errorf("backfill comic %d: %w", id, err)
			}
			if affected, _ := result.RowsAffected(); affected > 0 {
				inserted++
			}
		}
	}
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

// DropLegacyIdentityKeyColumn removes the deprecated comics.identity_key column (SQLite 3.35+).
// Run offline via maintenance; not invoked on API startup (can rewrite large tables).
func DropLegacyIdentityKeyColumn(db *sql.DB) error {
	return dropLegacyIdentityKeyColumn(db)
}

func dropLegacyIdentityKeyColumn(db *sql.DB) error {
	_, _ = db.Exec(`DROP INDEX IF EXISTS uq_comics_identity_key`)
	_, _ = db.Exec(`DROP INDEX IF EXISTS idx_comics_identity_key`)
	if !sqliteTableHasColumn(db, "comics", "identity_key") {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE comics DROP COLUMN identity_key`)
	return err
}

func sqliteTableHasColumn(db *sql.DB, table string, column string) bool {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

func discoveryScopeForTitle(title string, comType int) string {
	return identity.DiscoveryScope(title, comType)
}

func discoveryTitleKey(title string) string {
	return identity.DiscoveryTitleKey(title)
}

// SyncDiscoveryKeys replaces discovery keys for a comic from its title variants.
func (r *SQLComicRepository) SyncDiscoveryKeys(ctx context.Context, comicID int, comType int, titles []string) error {
	if err := r.DeleteDiscoveryKeysForComic(ctx, comicID); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, title := range titles {
		scope := discoveryScopeForTitle(title, comType)
		titleKey := discoveryTitleKey(title)
		if titleKey == "" {
			continue
		}
		scoped := scope + "\x00" + titleKey
		if _, ok := seen[scoped]; ok {
			continue
		}
		seen[scoped] = struct{}{}
		if err := r.insertDiscoveryKey(ctx, comicID, scope, titleKey); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLComicRepository) insertDiscoveryKey(ctx context.Context, comicID int, scope string, titleKey string) error {
	query := `INSERT INTO comic_title_keys (comic_id, scope, title_key) VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `)`
	if r.dialect == dialectSQLite {
		query = `INSERT OR IGNORE INTO comic_title_keys (comic_id, scope, title_key) VALUES (` + r.placeholder(1) + `, ` + r.placeholder(2) + `, ` + r.placeholder(3) + `)`
	} else {
		query += ` ON CONFLICT (scope, title_key) DO NOTHING`
	}
	_, err := r.exec(ctx, query, comicID, scope, titleKey)
	return err
}

func (r *SQLComicRepository) DeleteDiscoveryKeysForComic(ctx context.Context, comicID int) error {
	_, err := r.exec(ctx, `DELETE FROM comic_title_keys WHERE comic_id = `+r.placeholder(1), comicID)
	return err
}

// FindComicsByDiscoveryTitleKeys returns active comics matching any title_key in scope.
func (r *SQLComicRepository) FindComicsByDiscoveryTitleKeys(
	ctx context.Context,
	scope string,
	titleKeys []string,
) ([]domain.Comic, error) {
	if len(titleKeys) == 0 {
		return nil, nil
	}
	deletedFilter := "c.deleted = 0"
	if r.dialect == dialectPostgres {
		deletedFilter = "c.deleted = false"
	}
	placeholders := make([]string, len(titleKeys))
	args := make([]any, 0, len(titleKeys)+1)
	args = append(args, scope)
	for i, key := range titleKeys {
		placeholders[i] = r.placeholder(i + 2)
		args = append(args, key)
	}
	query := fmt.Sprintf(
		`SELECT DISTINCT %s FROM comics c
		 INNER JOIN comic_title_keys k ON k.comic_id = c.id
		 WHERE k.scope = %s AND k.title_key IN (%s) AND %s`,
		comicSelectColumns("c"),
		r.placeholder(1),
		strings.Join(placeholders, ", "),
		deletedFilter,
	)
	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComics(rows)
}

// FindComicsByDiscoveryKeyToken returns comics whose discovery title_key contains token.
func (r *SQLComicRepository) FindComicsByDiscoveryKeyToken(
	ctx context.Context,
	scope string,
	token string,
	limit int,
) ([]domain.Comic, error) {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	deletedFilter := "c.deleted = 0"
	likeOp := "LIKE"
	if r.dialect == dialectPostgres {
		deletedFilter = "c.deleted = false"
		likeOp = "ILIKE"
	}
	pattern := "%" + token + "%"
	query := fmt.Sprintf(
		`SELECT DISTINCT %s FROM comics c
		 INNER JOIN comic_title_keys k ON k.comic_id = c.id
		 WHERE k.scope = %s AND k.title_key %s %s AND %s
		 LIMIT %s`,
		comicSelectColumns("c"),
		r.placeholder(1),
		likeOp,
		r.placeholder(2),
		deletedFilter,
		r.placeholder(3),
	)
	rows, err := r.query(ctx, query, scope, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComics(rows)
}

// ListDiscoveryKeyCollisions returns comic ids sharing the same scoped title_key.
func (r *SQLComicRepository) ListDiscoveryKeyCollisions(ctx context.Context) ([]DiscoveryCollision, error) {
	deletedFilter := "c.deleted = 0"
	if r.dialect == dialectPostgres {
		deletedFilter = "c.deleted = false"
	}
	query := `SELECT k.scope, k.title_key, GROUP_CONCAT(DISTINCT k.comic_id)
		FROM comic_title_keys k
		INNER JOIN comics c ON c.id = k.comic_id
		WHERE ` + deletedFilter + `
		GROUP BY k.scope, k.title_key
		HAVING COUNT(DISTINCT k.comic_id) > 1`
	if r.dialect == dialectPostgres {
		query = `SELECT k.scope, k.title_key, STRING_AGG(DISTINCT k.comic_id::text, ',')
			FROM comic_title_keys k
			INNER JOIN comics c ON c.id = k.comic_id
			WHERE ` + deletedFilter + `
			GROUP BY k.scope, k.title_key
			HAVING COUNT(DISTINCT k.comic_id) > 1`
	}
	rows, err := r.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DiscoveryCollision
	for rows.Next() {
		var item DiscoveryCollision
		var ids string
		if err := rows.Scan(&item.Scope, &item.TitleKey, &ids); err != nil {
			return nil, err
		}
		for _, part := range strings.Split(ids, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var id int
			if _, err := fmt.Sscanf(part, "%d", &id); err == nil {
				item.ComicIDs = append(item.ComicIDs, id)
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type DiscoveryCollision struct {
	Scope    string
	TitleKey string
	ComicIDs []int
}

func comicSelectColumns(alias string) string {
	if alias == "" {
		alias = "comics"
	}
	return fmt.Sprintf(`%s.id, %s.titles, %s.current_chap, %s.cover, CAST(%s.last_update AS TEXT),
		%s.com_type, %s.status, %s.published_in, %s.genres, %s.description, %s.author, %s.track,
		%s.viewed_chap, %s.rating, %s.deleted, %s.cover_visible`, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias, alias)
}
