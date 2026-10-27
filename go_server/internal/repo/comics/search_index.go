package comics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"comics/domain"
)

const (
	catalogFTSTable = "comics_titles_fts"

	searchModeAuto    = "auto"
	searchModeScan    = "scan"
	searchModeTrigram = "trigram"
)

var catalogSearchDDL = []string{
	`CREATE VIRTUAL TABLE IF NOT EXISTS comics_titles_fts USING fts5(
		titles,
		content='comics',
		content_rowid='id',
		tokenize='trigram'
	)`,
	`CREATE TRIGGER IF NOT EXISTS comics_titles_fts_ai AFTER INSERT ON comics BEGIN
		INSERT INTO comics_titles_fts(rowid, titles) VALUES (new.id, new.titles);
	END`,
	`CREATE TRIGGER IF NOT EXISTS comics_titles_fts_ad AFTER DELETE ON comics BEGIN
		INSERT INTO comics_titles_fts(comics_titles_fts, rowid, titles) VALUES('delete', old.id, old.titles);
	END`,
	`CREATE TRIGGER IF NOT EXISTS comics_titles_fts_au AFTER UPDATE OF titles ON comics BEGIN
		INSERT INTO comics_titles_fts(comics_titles_fts, rowid, titles) VALUES('delete', old.id, old.titles);
		INSERT INTO comics_titles_fts(rowid, titles) VALUES (new.id, new.titles);
	END`,
}

// SearchModeFromEnv returns auto, scan, or trigram.
func SearchModeFromEnv() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("COMICS_SEARCH_MODE")))
	switch mode {
	case searchModeScan, searchModeTrigram, searchModeAuto:
		return mode
	default:
		return searchModeAuto
	}
}

// TrigramSearchEligible reports whether the term can use the FTS trigram index.
func TrigramSearchEligible(term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	if strings.Contains(term, "%") || strings.Contains(term, "_") {
		return false
	}
	runes := utf8.RuneCountInString(term)
	if runes < 3 {
		return false
	}
	return true
}

func ensureCatalogSearchSchema(ctx context.Context, db *sql.DB) error {
	for _, stmt := range catalogSearchDDL {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("catalog search schema: %w", err)
		}
	}
	return nil
}

func catalogSearchIndexHealthy(ctx context.Context, db *sql.DB) (bool, error) {
	var ftsRows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+catalogFTSTable).Scan(&ftsRows); err != nil {
		return false, err
	}
	var active int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comics WHERE deleted = 0`).Scan(&active); err != nil {
		return false, err
	}
	if active == 0 {
		return ftsRows == 0, nil
	}
	// Allow small drift during concurrent writes; empty FTS means rebuild required.
	return ftsRows > 0, nil
}

func (r *SQLComicRepository) trigramSearchEnabled(ctx context.Context) bool {
	if r.dialect != dialectSQLite {
		return false
	}
	r.trigramHealthOnce.Do(func() {
		r.trigramHealthOK = trigramFTSCapable(ctx, r.readConn())
		if !r.trigramHealthOK {
			return
		}
		ok, err := catalogSearchIndexHealthy(ctx, r.readConn())
		r.trigramHealthOK = err == nil && ok
	})
	if !r.trigramHealthOK {
		return false
	}
	mode := SearchModeFromEnv()
	switch mode {
	case searchModeScan, searchModeAuto:
		// Auto stays on the legacy comics.titles scan until FTS can accelerate that
		// predicate without changing pipe-delimited substring semantics.
		return false
	case searchModeTrigram:
		return true
	default:
		return false
	}
}

// RebuildSearchIndex rebuilds the FTS index using this repository's writer connection.
func (r *SQLComicRepository) RebuildSearchIndex(ctx context.Context) error {
	return RebuildCatalogSearchIndex(ctx, r.db)
}

// RebuildCatalogSearchIndex rebuilds the FTS index from the comics content table (offline maintenance).
func RebuildCatalogSearchIndex(ctx context.Context, db *sql.DB) error {
	if err := ensureCatalogSearchSchema(ctx, db); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `INSERT INTO `+catalogFTSTable+`(`+catalogFTSTable+`) VALUES('rebuild')`)
	return err
}

// CheckCatalogSearchIndex compares FTS row count to active comics.
func CheckCatalogSearchIndex(ctx context.Context, db *sql.DB) error {
	if err := ensureCatalogSearchSchema(ctx, db); err != nil {
		return err
	}
	ok, err := catalogSearchIndexHealthy(ctx, db)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("catalog search index is empty or unavailable; run rebuild-catalog-search")
	}
	return nil
}

func trigramFTSCapable(ctx context.Context, db *sql.DB) bool {
	if _, err := db.ExecContext(ctx, `CREATE VIRTUAL TABLE IF NOT EXISTS _fts_trigram_probe USING fts5(x, tokenize='trigram')`); err != nil {
		return false
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS _fts_trigram_probe`)
	return true
}

type searchFilterResult struct {
	joinSQL string
	where   string
	args    []any
	path    string // "trigram" or "scan"
}

func appendTitleScanPredicate(dialect string, baseWhere string, baseArgs []any, term string) (string, []any) {
	pattern := "%" + term + "%"
	args := append(append([]any{}, baseArgs...), pattern)
	var extra string
	if dialect == dialectPostgres {
		extra = fmt.Sprintf("titles ILIKE $%d", len(args))
	} else {
		extra = "titles LIKE ? COLLATE NOCASE"
	}
	where := baseWhere
	if strings.TrimSpace(where) == "" {
		where = "WHERE " + extra
	} else {
		where = where + " AND " + extra
	}
	return where, args
}

func (r *SQLComicRepository) buildTitleSearchFilter(
	ctx context.Context,
	query domain.ComicListQuery,
) searchFilterResult {
	baseWhere, baseArgs := r.comicFilters(query)
	if query.SearchTitle == "" {
		return searchFilterResult{where: baseWhere, args: baseArgs, path: "none"}
	}

	term := query.SearchTitle
	if SearchModeFromEnv() != searchModeTrigram ||
		!TrigramSearchEligible(term) ||
		!r.trigramSearchEnabled(ctx) {
		where, args := appendTitleScanPredicate(r.dialect, baseWhere, baseArgs, term)
		return searchFilterResult{where: where, args: args, path: "scan"}
	}

	// Join FTS for planner hints; substring semantics stay on comics.titles (pipe-delimited variants).
	pattern := "%" + term + "%"
	join := ` INNER JOIN ` + catalogFTSTable + ` fts ON fts.rowid = comics.id`
	extra := ` comics.titles LIKE ? COLLATE NOCASE`
	args := append(append([]any{}, baseArgs...), pattern)

	where := baseWhere
	if strings.TrimSpace(where) == "" {
		where = "WHERE " + extra
	} else {
		where = where + " AND " + extra
	}
	return searchFilterResult{
		joinSQL: join,
		where:   where,
		args:    args,
		path:    "trigram",
	}
}
