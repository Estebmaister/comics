package comics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

const (
	sqlInsertComicTitleKeySQLite = `INSERT OR IGNORE INTO comic_title_keys
		(comic_id, scope, title_key) VALUES (?, ?, ?)`
	sqlInsertComicTitleKeyPostgres = `INSERT INTO comic_title_keys
		(comic_id, scope, title_key) VALUES ($1, $2, $3)
		ON CONFLICT (scope, title_key) DO NOTHING`
	sqlDeleteComicTitleKeysByComicSQLite   = `DELETE FROM comic_title_keys WHERE comic_id = ?`
	sqlDeleteComicTitleKeysByComicPostgres = `DELETE FROM comic_title_keys WHERE comic_id = $1`
)

func insertComicTitleKeySQL(dialect string) string {
	if dialect == dialectSQLite {
		return sqlInsertComicTitleKeySQLite
	}
	return sqlInsertComicTitleKeyPostgres
}

func deleteComicTitleKeysByComicSQL(dialect string) string {
	if dialect == dialectSQLite {
		return sqlDeleteComicTitleKeysByComicSQLite
	}
	return sqlDeleteComicTitleKeysByComicPostgres
}

func ensureComicTitleKeysSchema(db *sql.DB) error {
	_, err := db.Exec(comicTitleKeysDDL)
	return err
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
	if err != nil {
		return discoveryKeyOwner{}, false, err
	}
	return owner, true, nil
}

func (r *SQLComicRepository) deleteDiscoveryKeyForComic(
	ctx context.Context,
	comicID int,
	scope string,
	titleKey string,
) error {
	_, err := r.exec(
		ctx,
		`DELETE FROM comic_title_keys WHERE scope = `+r.placeholder(1)+
			` AND title_key = `+r.placeholder(2)+
			` AND comic_id = `+r.placeholder(3),
		scope,
		titleKey,
		comicID,
	)
	return err
}

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
		if err := r.deleteDiscoveryKeyForComic(ctx, owner.ComicID, scope, titleKey); err != nil {
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

// SyncDiscoveryKeys replaces discovery keys for a comic from its title variants.
func (r *SQLComicRepository) SyncDiscoveryKeys(ctx context.Context, comicID int, comType int, titles []string) error {
	if r.tx == nil {
		return r.WithTx(ctx, func(txRepo domain.ComicRepository) error {
			sqlRepo, ok := txRepo.(*SQLComicRepository)
			if !ok {
				return fmt.Errorf("unexpected comic repository in transaction")
			}
			return sqlRepo.SyncDiscoveryKeys(ctx, comicID, comType, titles)
		})
	}

	pairs := identity.DiscoveryTitleKeyPairsForTitles(titles, comType)
	for _, pair := range pairs {
		owner, found, err := r.discoveryKeyOwner(ctx, pair.Scope, pair.TitleKey)
		if err != nil {
			return err
		}
		if !found || owner.ComicID == comicID {
			continue
		}
		if owner.Deleted {
			if err := r.deleteDiscoveryKeyForComic(ctx, owner.ComicID, pair.Scope, pair.TitleKey); err != nil {
				return err
			}
			continue
		}
		return &domain.DiscoveryKeyConflictError{
			ComicID:      comicID,
			OwnerComicID: owner.ComicID,
			Scope:        pair.Scope,
			TitleKey:     pair.TitleKey,
		}
	}

	if err := r.DeleteDiscoveryKeysForComic(ctx, comicID); err != nil {
		return err
	}
	for _, pair := range pairs {
		if err := r.insertDiscoveryKey(ctx, comicID, pair.Scope, pair.TitleKey); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLComicRepository) DeleteDiscoveryKeysForComic(ctx context.Context, comicID int) error {
	_, err := r.exec(ctx, deleteComicTitleKeysByComicSQL(r.dialect), comicID)
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
