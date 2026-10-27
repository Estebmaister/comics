package comics

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"comics/domain"
	"comics/internal/discovery/repair"
	"comics/internal/identity"
)

func (r *SQLComicRepository) AuditDiscoveryIntegrity(ctx context.Context) (repair.Audit, error) {
	if err := r.checkSQLiteIntegrity(ctx); err != nil {
		return repair.Audit{}, err
	}

	summary, err := r.discoveryAuditSummary(ctx)
	if err != nil {
		return repair.Audit{}, err
	}

	fingerprint, err := r.discoveryDatabaseFingerprint(ctx)
	if err != nil {
		return repair.Audit{}, err
	}

	keyless, err := r.listActiveKeylessComics(ctx)
	if err != nil {
		return repair.Audit{}, err
	}

	records := make([]repair.KeylessRecord, 0, len(keyless))
	for _, comic := range keyless {
		record, err := r.buildKeylessRecord(ctx, comic)
		if err != nil {
			return repair.Audit{}, err
		}
		records = append(records, record)
		switch record.Classification {
		case repair.ClassificationActiveConflict:
			summary.ActiveConflicts++
		case repair.ClassificationRestorable:
			summary.Restorable++
		case repair.ClassificationBlockedNoKey:
			summary.BlockedNoKey++
		}
	}
	summary.KeylessActive = len(records)

	return repair.Audit{
		DatabaseFingerprint: fingerprint,
		Summary:             summary,
		Records:             records,
	}, nil
}

func (r *SQLComicRepository) checkSQLiteIntegrity(ctx context.Context) error {
	if r.dialect != dialectSQLite {
		return nil
	}
	var result string
	if err := r.queryRow(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("sqlite integrity_check failed: %s", result)
	}
	return nil
}

func (r *SQLComicRepository) discoveryAuditSummary(ctx context.Context) (repair.AuditSummary, error) {
	var summary repair.AuditSummary
	err := r.queryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM comics),
			(SELECT COUNT(*) FROM comics WHERE deleted = 0),
			(SELECT COUNT(*) FROM comic_title_keys),
			(SELECT COUNT(DISTINCT k.comic_id)
			 FROM comic_title_keys k
			 INNER JOIN comics c ON c.id = k.comic_id
			 WHERE c.deleted = 0),
			(SELECT COUNT(*)
			 FROM comic_title_keys k
			 INNER JOIN comics c ON c.id = k.comic_id
			 WHERE c.deleted = 1),
			(SELECT COUNT(*)
			 FROM comic_title_keys k
			 LEFT JOIN comics c ON c.id = k.comic_id
			 WHERE c.id IS NULL)`).Scan(
		&summary.TotalComics,
		&summary.ActiveComics,
		&summary.KeyRows,
		&summary.ActiveKeyOwners,
		&summary.DeletedKeyRows,
		&summary.OrphanKeyRows,
	)
	return summary, err
}

func (r *SQLComicRepository) discoveryDatabaseFingerprint(ctx context.Context) (string, error) {
	rows, err := r.query(ctx, `
		SELECT id, titles, current_chap, com_type, track, deleted
		FROM comics
		ORDER BY id`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	hash := sha256.New()
	for rows.Next() {
		var id, chap, comType int
		var titles string
		var track, deleted bool
		if err := rows.Scan(&id, &titles, &chap, &comType, &track, &deleted); err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "comic\x00%d\x00%s\x00%d\x00%d\x00%t\x00%t\n",
			id, titles, chap, comType, track, deleted)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	keyRows, err := r.query(ctx, `
		SELECT comic_id, scope, title_key
		FROM comic_title_keys
		ORDER BY scope, title_key, comic_id`)
	if err != nil {
		return "", err
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var comicID int
		var scope, titleKey string
		if err := keyRows.Scan(&comicID, &scope, &titleKey); err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "key\x00%d\x00%s\x00%s\n", comicID, scope, titleKey)
	}
	if err := keyRows.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (r *SQLComicRepository) listActiveKeylessComics(ctx context.Context) ([]domain.Comic, error) {
	query := `SELECT ` + comicSelectColumns("c") + `
		FROM comics c
		WHERE c.deleted = 0
		  AND NOT EXISTS (
			SELECT 1 FROM comic_title_keys k WHERE k.comic_id = c.id
		  )
		ORDER BY c.id`
	rows, err := r.query(ctx, query)
	if err != nil {
		return nil, err
	}
	return scanComics(rows)
}

func (r *SQLComicRepository) buildKeylessRecord(ctx context.Context, comic domain.Comic) (repair.KeylessRecord, error) {
	titles := identity.NormalizeTitleVariants(comic.Titles, comic.ComType)
	proposed := identity.DiscoveryTitleKeyPairsForTitles(titles, comic.ComType)

	record := repair.KeylessRecord{
		Comic:        toComicSnapshot(comic),
		ProposedKeys: proposed,
	}

	if len(proposed) == 0 {
		record.Classification = repair.ClassificationBlockedNoKey
		return record, nil
	}

	ownerByID := map[int]repair.ComicSnapshot{}
	owners := make([]repair.KeyOwner, 0)
	activeConflict := false

	for _, key := range proposed {
		comics, err := r.FindComicsByDiscoveryTitleKeys(ctx, key.Scope, []string{key.TitleKey})
		if err != nil {
			return repair.KeylessRecord{}, err
		}
		for _, owner := range comics {
			if owner.ID == comic.ID {
				continue
			}
			if !owner.Deleted {
				activeConflict = true
			}
			if _, seen := ownerByID[owner.ID]; seen {
				continue
			}
			ownerByID[owner.ID] = toComicSnapshot(owner)
			owners = append(owners, repair.KeyOwner{
				Key:   key,
				Comic: toComicSnapshot(owner),
			})
		}
	}

	sort.Slice(owners, func(i, j int) bool {
		if owners[i].Key.Scope != owners[j].Key.Scope {
			return owners[i].Key.Scope < owners[j].Key.Scope
		}
		if owners[i].Key.TitleKey != owners[j].Key.TitleKey {
			return owners[i].Key.TitleKey < owners[j].Key.TitleKey
		}
		return owners[i].Comic.ID < owners[j].Comic.ID
	})
	record.Owners = owners

	if activeConflict {
		record.Classification = repair.ClassificationActiveConflict
	} else {
		record.Classification = repair.ClassificationRestorable
	}
	return record, nil
}

func toComicSnapshot(comic domain.Comic) repair.ComicSnapshot {
	titles := append([]string(nil), comic.Titles...)
	return repair.ComicSnapshot{
		ID:          comic.ID,
		Titles:      titles,
		CurrentChap: comic.CurrentChap,
		ComType:     comic.ComType,
		Track:       comic.Track,
		Deleted:     comic.Deleted,
	}
}

func NewSQLiteComicRepositoryReadOnly(path string) (*SQLComicRepository, error) {
	if strings.TrimSpace(path) == "" {
		path = findSQLiteComicDB()
	}
	if path == "" {
		return nil, fmt.Errorf("comic sqlite database not found")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(absPath) + "?mode=ro&_pragma=query_only%3d1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := configureSQLiteReader(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLComicRepository{
		db:      db,
		readDB:  db,
		dialect: dialectSQLite,
		ownsDB:  true,
	}, nil
}
