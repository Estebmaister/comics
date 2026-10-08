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

	"comics/domain"
	"comics/internal/identity"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	identity_key TEXT NOT NULL DEFAULT '',
	cover_visible BOOLEAN NOT NULL DEFAULT true
);

ALTER SEQUENCE comic_id_seq OWNED BY comics.id;
ALTER TABLE comics ALTER COLUMN id SET DEFAULT nextval('comic_id_seq');

CREATE UNIQUE INDEX IF NOT EXISTS uq_comics_identity_key
	ON comics(identity_key)
	WHERE deleted = false AND identity_key <> '';

CREATE INDEX IF NOT EXISTS idx_comics_identity_key ON comics(identity_key);
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
	tx      *sql.Tx
	dialect string
	ownsDB  bool
}

func NewSQLiteComicRepository(path string) (*SQLComicRepository, error) {
	if path == "" {
		path = findSQLiteComicDB()
	}
	if path == "" {
		return nil, fmt.Errorf("comic sqlite database not found")
	}
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := ensureSQLitePerformanceIndexes(db); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLComicRepository{db: db, dialect: dialectSQLite, ownsDB: true}, nil
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
	_, err := db.Exec(postgresComicSchema)
	return err
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
		filepath.Join("..", "src", "db", "comics.db"),
		filepath.Join("src", "db", "comics.db"),
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
	if !r.ownsDB || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *SQLComicRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *SQLComicRepository) WithTx(ctx context.Context, fn func(domain.ComicRepository) error) error {
	if r.tx != nil {
		return fn(r)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txRepo := &SQLComicRepository{db: r.db, tx: tx, dialect: r.dialect}
	if err := fn(txRepo); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *SQLComicRepository) List(ctx context.Context, query domain.ComicListQuery) (domain.ComicListResult, error) {
	where, args := r.comicFilters(query)
	var total int
	if err := r.queryRow(ctx, "SELECT COUNT(*) FROM comics "+where, args...).Scan(&total); err != nil {
		return domain.ComicListResult{}, err
	}

	sqlQuery := baseComicSelect() + " " + where + " " + r.orderByClause(query)
	queryArgs := append([]any{}, args...)
	if query.Limit > 0 {
		sqlQuery += r.limitOffsetClause(len(queryArgs) + 1)
		queryArgs = append(queryArgs, query.Limit, query.Offset)
	}

	rows, err := r.query(ctx, sqlQuery, queryArgs...)
	if err != nil {
		return domain.ComicListResult{}, err
	}
	defer rows.Close()

	comics, err := scanComics(rows)
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

func (r *SQLComicRepository) GetByIdentityKey(ctx context.Context, identityKey string) (domain.Comic, error) {
	if strings.TrimSpace(identityKey) == "" {
		return domain.Comic{}, domain.ErrComicNotFound
	}
	deletedFilter := "deleted = 0"
	if r.dialect == dialectPostgres {
		deletedFilter = "deleted = false"
	}
	row := r.queryRow(
		ctx,
		baseComicSelect()+" WHERE identity_key = "+r.placeholder(1)+" AND "+deletedFilter+" LIMIT 1",
		identityKey,
	)
	comic, err := scanComic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Comic{}, domain.ErrComicNotFound
	}
	return comic, err
}

func (r *SQLComicRepository) Create(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	now := time.Now().Unix()
	if r.dialect == dialectPostgres {
		var id int
		err := r.queryRow(
			ctx,
			`INSERT INTO comics (
				titles, current_chap, cover, last_update, com_type, status,
				published_in, genres, description, author, track, viewed_chap,
				rating, deleted, cover_visible, identity_key
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
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
			comicIdentityKey(comic),
		).Scan(&id)
		if err != nil {
			return domain.Comic{}, err
		}
		return r.Get(ctx, id)
	}

	result, err := r.exec(
		ctx,
		`INSERT INTO comics (
			titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap,
			rating, deleted, cover_visible, identity_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		comicIdentityKey(comic),
	)
	if err != nil {
		return domain.Comic{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Comic{}, err
	}
	return r.Get(ctx, int(id))
}

func (r *SQLComicRepository) Update(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	now := time.Now().Unix()
	if r.dialect == dialectPostgres {
		result, err := r.exec(
			ctx,
			`UPDATE comics SET titles = $1, current_chap = $2, cover = $3, last_update = $4,
				com_type = $5, status = $6, published_in = $7, genres = $8, description = $9,
				author = $10, track = $11, viewed_chap = $12, rating = $13, deleted = $14,
				cover_visible = $15, identity_key = $16
			WHERE id = $17`,
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
			comic.CoverVisible,
			comicIdentityKey(comic),
			comic.ID,
		)
		if err != nil {
			return domain.Comic{}, err
		}
		if err := ensureRowsAffected(result); err != nil {
			return domain.Comic{}, err
		}
		return r.Get(ctx, comic.ID)
	}

	result, err := r.exec(
		ctx,
		`UPDATE comics SET titles = ?, current_chap = ?, cover = ?, last_update = ?,
			com_type = ?, status = ?, published_in = ?, genres = ?, description = ?,
			author = ?, track = ?, viewed_chap = ?, rating = ?, deleted = ?,
			cover_visible = ?, identity_key = ?
		WHERE id = ?`,
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
		comic.CoverVisible,
		comicIdentityKey(comic),
		comic.ID,
	)
	if err != nil {
		return domain.Comic{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return domain.Comic{}, err
	}
	return r.Get(ctx, comic.ID)
}

func (r *SQLComicRepository) Delete(ctx context.Context, id int) error {
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
		if query.SearchTitle != "" {
			args = append(args, "%"+query.SearchTitle+"%")
			filters = append(filters, fmt.Sprintf("titles ILIKE $%d", len(args)))
		}
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
	if query.SearchTitle != "" {
		filters = append(filters, "titles LIKE ? COLLATE NOCASE")
		args = append(args, "%"+query.SearchTitle+"%")
	}
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

func (r *SQLComicRepository) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if r.tx != nil {
		return r.tx.QueryContext(ctx, query, args...)
	}
	return r.db.QueryContext(ctx, query, args...)
}

func (r *SQLComicRepository) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if r.tx != nil {
		return r.tx.QueryRowContext(ctx, query, args...)
	}
	return r.db.QueryRowContext(ctx, query, args...)
}

func (r *SQLComicRepository) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if r.tx != nil {
		return r.tx.ExecContext(ctx, query, args...)
	}
	return r.db.ExecContext(ctx, query, args...)
}

func baseComicSelect() string {
	return `SELECT id, titles, current_chap, cover, CAST(last_update AS TEXT),
		com_type, status, published_in, genres, description, author, track,
		viewed_chap, rating, deleted, cover_visible FROM comics`
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

func comicIdentityKey(comic domain.Comic) string {
	return identity.BuildIdentityKeyFromTitles(comic.Titles, comic.ComType)
}
