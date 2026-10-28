package comics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sync"

	"comics/domain"
	"comics/internal/identity"
	"comics/internal/metrics"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver for postgres DSNs
	_ "modernc.org/sqlite"
)

const (
	dialectSQLite       = "sqlite"
	dialectPostgres     = "postgres"
	sqliteBusyTimeoutMS = 30000
	postgresComicSchema = `
CREATE SEQUENCE IF NOT EXISTS comic_id_seq;

CREATE TABLE IF NOT EXISTS comics (
	id INTEGER PRIMARY KEY DEFAULT nextval('comic_id_seq'),
	titles TEXT NOT NULL,
	current_chap INTEGER NOT NULL DEFAULT 0,
	cover TEXT NOT NULL DEFAULT '',
	last_update BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM now())::BIGINT,
	com_type INTEGER NOT NULL DEFAULT 0,
	status INTEGER NOT NULL DEFAULT 0,
	published_in TEXT NOT NULL DEFAULT '',
	genres TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	author TEXT NOT NULL DEFAULT '',
	track BOOLEAN NOT NULL DEFAULT false,
	viewed_chap INTEGER NOT NULL DEFAULT 0,
	rating INTEGER NOT NULL DEFAULT 0,
	deleted BOOLEAN NOT NULL DEFAULT false,
	cover_visible BOOLEAN NOT NULL DEFAULT true
);

ALTER SEQUENCE comic_id_seq OWNED BY comics.id;
ALTER TABLE comics ALTER COLUMN id SET DEFAULT nextval('comic_id_seq');

CREATE INDEX IF NOT EXISTS idx_comics_active_update ON comics(deleted, last_update DESC, id);
CREATE INDEX IF NOT EXISTS idx_comics_active_track_update ON comics(deleted, track, last_update DESC, id);
`
)

var sqlitePerformanceIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_comics_active_update
		ON comics(deleted, last_update DESC, id)`,
	`CREATE INDEX IF NOT EXISTS idx_comics_active_track_update
		ON comics(deleted, track, last_update DESC, id)`,
}

type SQLComicRepository struct {
	db      *sql.DB
	readDB  *sql.DB
	tx      *sql.Tx
	dialect string
	ownsDB  bool

	trigramHealthOnce sync.Once
	trigramHealthOK   bool
}

func NewSQLiteComicRepository(path string) (*SQLComicRepository, error) {
	if path == "" {
		path = findSQLiteComicDB()
	}
	if path == "" {
		return nil, fmt.Errorf("comic sqlite database not found")
	}
	writer, reader, err := openSQLitePools(path)
	if err != nil {
		return nil, err
	}
	if err := ensureSQLitePerformanceIndexes(writer); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, err
	}
	if err := ensureComicTitleKeysSchema(writer); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, err
	}
	if SearchModeFromEnv() != searchModeScan {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := ensureCatalogSearchSchema(ctx, writer); err != nil {
			_ = writer.Close()
			_ = reader.Close()
			return nil, err
		}
	}
	return &SQLComicRepository{
		db:      writer,
		readDB:  reader,
		dialect: dialectSQLite,
		ownsDB:  true,
	}, nil
}

func openSQLitePools(path string) (*sql.DB, *sql.DB, error) {
	writer, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, nil, err
	}
	if err := writer.Ping(); err != nil {
		_ = writer.Close()
		return nil, nil, err
	}
	if err := configureSQLiteWriter(writer); err != nil {
		_ = writer.Close()
		return nil, nil, err
	}

	reader, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		_ = writer.Close()
		return nil, nil, err
	}
	if err := reader.Ping(); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, nil, err
	}
	if err := configureSQLiteReader(reader); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return nil, nil, err
	}
	return writer, reader, nil
}

func NewPostgresComicRepository(url string) (*SQLComicRepository, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("comic postgres url not found")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := EnsurePostgresComicSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLComicRepository{db: db, dialect: dialectPostgres, ownsDB: true}, nil
}

func EnsurePostgresComicSchema(db *sql.DB) error {
	if _, err := db.Exec(postgresComicSchema); err != nil {
		return err
	}
	return ensureComicTitleKeysSchema(db)
}

func sqliteDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf(
		"%s%s_pragma=journal_mode%%3dWAL&_pragma=busy_timeout%%3d%d",
		path,
		separator,
		sqliteBusyTimeoutMS,
	)
}

func ensureSQLitePerformanceIndexes(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range sqlitePerformanceIndexes {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func findSQLiteComicDB() string {
	candidates := []string{
		os.Getenv("COMICS_SQLITE_PATH"),
		filepath.Join("..", "db", "comics.db"),
		filepath.Join("db", "comics.db"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (r *SQLComicRepository) Close() error {
	if !r.ownsDB {
		return nil
	}
	var firstErr error
	if r.readDB != nil && r.readDB != r.db {
		if err := r.readDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if r.dialect == dialectSQLite && r.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := r.CheckpointWAL(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if r.db != nil {
		if err := r.db.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *SQLComicRepository) Ping(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return err
	}
	if r.dialect == dialectSQLite && SearchModeFromEnv() == searchModeTrigram {
		return CheckCatalogSearchIndex(ctx, r.db)
	}
	return nil
}

func (r *SQLComicRepository) WithTx(ctx context.Context, fn func(domain.ComicRepository) error) error {
	if r.tx != nil {
		return fn(r)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txRepo := &SQLComicRepository{db: r.db, readDB: r.readDB, tx: tx, dialect: r.dialect}
	if err := fn(txRepo); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *SQLComicRepository) List(ctx context.Context, query domain.ComicListQuery) (domain.ComicListResult, error) {
	search := r.buildTitleSearchFilter(ctx, query)
	if query.SearchTitle != "" {
		metrics.IncCatalogSearchPath(search.path)
	}
	fromClause := "comics" + search.joinSQL
	where, args := search.where, search.args
	var total int
	countStart := time.Now()
	if err := r.queryRow(ctx, "SELECT COUNT(*) FROM "+fromClause+" "+where, args...).Scan(&total); err != nil {
		return domain.ComicListResult{}, err
	}
	metrics.ObserveCatalogRepo("count", search.path, time.Since(countStart))

	sqlQuery := baseComicSelectFrom(fromClause) + " " + where + " " + r.orderByClause(query)
	queryArgs := append([]any{}, args...)
	if query.Limit > 0 {
		sqlQuery += r.limitOffsetClause(len(queryArgs) + 1)
		queryArgs = append(queryArgs, query.Limit, query.Offset)
	}

	pageStart := time.Now()
	rows, err := r.query(ctx, sqlQuery, queryArgs...)
	if err != nil {
		return domain.ComicListResult{}, err
	}
	defer rows.Close()

	comics, err := scanComics(rows)
	metrics.ObserveCatalogRepo("page", search.path, time.Since(pageStart))
	if err != nil {
		return domain.ComicListResult{}, err
	}

	totalPages := 1
	currentPage := 1
	if query.Limit > 0 {
		totalPages = (total + query.Limit - 1) / query.Limit
		currentPage = query.Offset/query.Limit + 1
	}
	return domain.ComicListResult{
		Comics:      comics,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: currentPage,
	}, nil
}

func (r *SQLComicRepository) Get(ctx context.Context, id int) (domain.Comic, error) {
	row := r.queryRow(ctx, baseComicSelect()+" WHERE id = "+r.placeholder(1), id)
	comic, err := scanComic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Comic{}, domain.ErrComicNotFound
	}
	return comic, err
}

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

func (r *SQLComicRepository) syncDiscoveryKeysAfterWrite(
	ctx context.Context,
	comic domain.Comic,
) error {
	if comic.Deleted {
		return r.DeleteDiscoveryKeysForComic(ctx, comic.ID)
	}
	return r.SyncDiscoveryKeys(ctx, comic.ID, comic.ComType, comic.Titles)
}

func (r *SQLComicRepository) Create(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
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

func (r *SQLComicRepository) create(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	comic, err := normalizeComicForWrite(comic)
	if err != nil {
		return domain.Comic{}, err
	}
	now := time.Now().Unix()
	if r.dialect == dialectPostgres {
		var id int
		err := r.queryRow(
			ctx,
			`INSERT INTO comics (
				titles, current_chap, cover, last_update, com_type, status,
				published_in, genres, description, author, track, viewed_chap,
				rating, deleted, cover_visible
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			RETURNING id`,
			strings.Join(comic.Titles, "|"),
			comic.CurrentChap,
			comic.Cover,
			now,
			comic.ComType,
			comic.Status,
			joinInts(comic.PublishedIn),
			joinInts(comic.Genres),
			comic.Description,
			comic.Author,
			comic.Track,
			comic.ViewedChap,
			comic.Rating,
			comic.Deleted,
			coverVisibleOrDefault(comic),
		).Scan(&id)
		if err != nil {
			return domain.Comic{}, err
		}
		comic.ID = id
		if err := r.syncDiscoveryKeysAfterWrite(ctx, comic); err != nil {
			return domain.Comic{}, err
		}
		return r.Get(ctx, id)
	}

	result, err := r.exec(
		ctx,
		`INSERT INTO comics (
			titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap,
			rating, deleted, cover_visible
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.Join(comic.Titles, "|"),
		comic.CurrentChap,
		comic.Cover,
		now,
		comic.ComType,
		comic.Status,
		joinInts(comic.PublishedIn),
		joinInts(comic.Genres),
		comic.Description,
		comic.Author,
		boolInt(comic.Track),
		comic.ViewedChap,
		comic.Rating,
		boolInt(comic.Deleted),
		coverVisibleOrDefault(comic),
	)
	if err != nil {
		return domain.Comic{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Comic{}, err
	}
	comic.ID = int(id)
	if err := r.syncDiscoveryKeysAfterWrite(ctx, comic); err != nil {
		return domain.Comic{}, err
	}
	return r.Get(ctx, comic.ID)
}

func (r *SQLComicRepository) Update(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
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

func (r *SQLComicRepository) update(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	comic, err := normalizeComicForWrite(comic)
	if err != nil {
		return domain.Comic{}, err
	}
	lastUpdate := comicLastUpdateUnix(comic)
	if r.dialect == dialectPostgres {
		result, err := r.exec(
			ctx,
			`UPDATE comics SET titles = $1, current_chap = $2, cover = $3, last_update = $4,
				com_type = $5, status = $6, published_in = $7, genres = $8, description = $9,
				author = $10, track = $11, viewed_chap = $12, rating = $13, deleted = $14,
				cover_visible = $15
			WHERE id = $16`,
			strings.Join(comic.Titles, "|"),
			comic.CurrentChap,
			comic.Cover,
			lastUpdate,
			comic.ComType,
			comic.Status,
			joinInts(comic.PublishedIn),
			joinInts(comic.Genres),
			comic.Description,
			comic.Author,
			comic.Track,
			comic.ViewedChap,
			comic.Rating,
			comic.Deleted,
			comic.CoverVisible,
			comic.ID,
		)
		if err != nil {
			return domain.Comic{}, err
		}
		if err := ensureRowsAffected(result); err != nil {
			return domain.Comic{}, err
		}
		if err := r.syncDiscoveryKeysAfterWrite(ctx, comic); err != nil {
			return domain.Comic{}, err
		}
		return r.Get(ctx, comic.ID)
	}

	result, err := r.exec(
		ctx,
		`UPDATE comics SET titles = ?, current_chap = ?, cover = ?, last_update = ?,
			com_type = ?, status = ?, published_in = ?, genres = ?, description = ?,
			author = ?, track = ?, viewed_chap = ?, rating = ?, deleted = ?,
			cover_visible = ?
		WHERE id = ?`,
		strings.Join(comic.Titles, "|"),
		comic.CurrentChap,
		comic.Cover,
		lastUpdate,
		comic.ComType,
		comic.Status,
		joinInts(comic.PublishedIn),
		joinInts(comic.Genres),
		comic.Description,
		comic.Author,
		boolInt(comic.Track),
		comic.ViewedChap,
		comic.Rating,
		boolInt(comic.Deleted),
		comic.CoverVisible,
		comic.ID,
	)
	if err != nil {
		return domain.Comic{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return domain.Comic{}, err
	}
	if err := r.syncDiscoveryKeysAfterWrite(ctx, comic); err != nil {
		return domain.Comic{}, err
	}
	return r.Get(ctx, comic.ID)
}

func (r *SQLComicRepository) Delete(ctx context.Context, id int) error {
	if r.tx != nil {
		return r.delete(ctx, id)
	}
	return r.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		return txRepo.Delete(ctx, id)
	})
}

func (r *SQLComicRepository) delete(ctx context.Context, id int) error {
	if err := r.DeleteDiscoveryKeysForComic(ctx, id); err != nil {
		return err
	}
	result, err := r.exec(ctx, "DELETE FROM comics WHERE id = "+r.placeholder(1), id)
	if err != nil {
		return err
	}
	return ensureRowsAffected(result)
}

func (r *SQLComicRepository) SetCoverVisibility(ctx context.Context, id int, visible bool) (domain.Comic, error) {
	var visibleValue any = visible
	if r.dialect == dialectSQLite {
		visibleValue = boolInt(visible)
	}
	result, err := r.exec(
		ctx,
		"UPDATE comics SET cover_visible = "+r.placeholder(1)+" WHERE id = "+r.placeholder(2),
		visibleValue,
		id,
	)
	if err != nil {
		return domain.Comic{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return domain.Comic{}, err
	}
	return r.Get(ctx, id)
}

func ensureRowsAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrComicNotFound
	}
	return nil
}

func (r *SQLComicRepository) comicFilters(query domain.ComicListQuery) (string, []any) {
	args := []any{}
	filters := []string{}
	if r.dialect == dialectPostgres {
		filters = append(filters, "deleted = false")
		if query.OnlyTracked {
			filters = append(filters, "track = true")
		}
		if query.OnlyUnchecked {
			filters = append(filters, "track = true", "current_chap != viewed_chap")
		}
		if query.RatingMin != nil {
			args = append(args, *query.RatingMin)
			filters = append(filters, fmt.Sprintf("rating >= $%d", len(args)))
		}
		if query.RatingMax != nil {
			args = append(args, *query.RatingMax)
			filters = append(filters, fmt.Sprintf("rating <= $%d", len(args)))
		}
		return "WHERE " + strings.Join(filters, " AND "), args
	}

	filters = append(filters, "deleted = 0")
	if query.OnlyTracked {
		filters = append(filters, "track = 1")
	}
	if query.OnlyUnchecked {
		filters = append(filters, "track = 1", "current_chap != viewed_chap")
	}
	if query.RatingMin != nil {
		filters = append(filters, "rating >= ?")
		args = append(args, *query.RatingMin)
	}
	if query.RatingMax != nil {
		filters = append(filters, "rating <= ?")
		args = append(args, *query.RatingMax)
	}
	return "WHERE " + strings.Join(filters, " AND "), args
}

func (r *SQLComicRepository) orderByClause(query domain.ComicListQuery) string {
	column := "last_update"
	switch strings.ToLower(strings.TrimSpace(query.SortBy)) {
	case "rating":
		column = "rating"
	case "id":
		column = "id"
	case "last_update", "":
		column = "last_update"
	}

	direction := "DESC"
	if strings.EqualFold(strings.TrimSpace(query.SortDir), "asc") {
		direction = "ASC"
	}

	switch column {
	case "rating":
		return fmt.Sprintf("ORDER BY rating %s, last_update DESC, id", direction)
	case "id":
		return fmt.Sprintf("ORDER BY id %s", direction)
	default:
		return fmt.Sprintf("ORDER BY last_update %s, id", direction)
	}
}

func (r *SQLComicRepository) limitOffsetClause(start int) string {
	if r.dialect == dialectPostgres {
		return fmt.Sprintf(" LIMIT $%d OFFSET $%d", start, start+1)
	}
	return " LIMIT ? OFFSET ?"
}

func (r *SQLComicRepository) placeholder(index int) string {
	if r.dialect == dialectPostgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func (r *SQLComicRepository) readConn() *sql.DB {
	if r.readDB != nil {
		return r.readDB
	}
	return r.db
}

func (r *SQLComicRepository) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if r.tx != nil {
		return r.tx.QueryContext(ctx, query, args...)
	}
	return r.readConn().QueryContext(ctx, query, args...)
}

func (r *SQLComicRepository) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if r.tx != nil {
		return r.tx.QueryRowContext(ctx, query, args...)
	}
	return r.readConn().QueryRowContext(ctx, query, args...)
}

func (r *SQLComicRepository) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if r.tx != nil {
		return r.tx.ExecContext(ctx, query, args...)
	}
	return r.db.ExecContext(ctx, query, args...)
}

func baseComicSelect() string {
	return baseComicSelectFrom("comics")
}

func baseComicSelectFrom(fromClause string) string {
	return `SELECT comics.id, comics.titles, comics.current_chap, comics.cover, CAST(comics.last_update AS TEXT),
		comics.com_type, comics.status, comics.published_in, comics.genres, comics.description, comics.author, comics.track,
		comics.viewed_chap, comics.rating, comics.deleted, comics.cover_visible FROM ` + fromClause
}

type comicScanner interface {
	Scan(dest ...any) error
}

func scanComics(rows *sql.Rows) ([]domain.Comic, error) {
	comics := []domain.Comic{}
	for rows.Next() {
		comic, err := scanComic(rows)
		if err != nil {
			return nil, err
		}
		comics = append(comics, comic)
	}
	return comics, rows.Err()
}

func scanComic(row comicScanner) (domain.Comic, error) {
	var titles, lastUpdate, publishedIn, genres string
	var comic domain.Comic
	if err := row.Scan(
		&comic.ID,
		&titles,
		&comic.CurrentChap,
		&comic.Cover,
		&lastUpdate,
		&comic.ComType,
		&comic.Status,
		&publishedIn,
		&genres,
		&comic.Description,
		&comic.Author,
		&comic.Track,
		&comic.ViewedChap,
		&comic.Rating,
		&comic.Deleted,
		&comic.CoverVisible,
	); err != nil {
		return domain.Comic{}, err
	}
	comic.Titles = splitTitles(titles)
	comic.PublishedIn = splitInts(publishedIn)
	comic.Genres = splitInts(genres)
	comic.LastUpdate = parseLastUpdate(lastUpdate)
	return comic, nil
}

func splitTitles(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(value, "|")
}

func splitInts(value string) []int {
	if value == "" {
		return []int{}
	}
	parts := strings.Split(value, "|")
	values := []int{}
	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			values = append(values, parsed)
		}
	}
	return values
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, "|")
}

func parseLastUpdate(value string) time.Time {
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func comicLastUpdateUnix(comic domain.Comic) int64 {
	if comic.LastUpdate.IsZero() {
		return time.Now().Unix()
	}
	return comic.LastUpdate.Unix()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func coverVisibleOrDefault(comic domain.Comic) bool {
	if comic.Cover == "" {
		return true
	}
	return comic.CoverVisible
}
