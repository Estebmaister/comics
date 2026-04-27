package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrComicNotFound = errors.New("comic not found")

type ComicJSON struct {
	ID           int      `json:"id"`
	Titles       []string `json:"titles"`
	CurrentChap  int      `json:"current_chap"`
	Cover        string   `json:"cover"`
	CoverVisible bool     `json:"cover_visible"`
	LastUpdate   string   `json:"last_update"`
	ComType      int      `json:"com_type"`
	Status       int      `json:"status"`
	PublishedIn  []int    `json:"published_in"`
	Genres       []int    `json:"genres"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	Track        bool     `json:"track"`
	ViewedChap   int      `json:"viewed_chap"`
	Rating       int      `json:"rating"`
	Deleted      bool     `json:"deleted"`
}

type ComicListResult struct {
	Comics      []ComicJSON
	Total       int
	TotalPages  int
	CurrentPage int
}

type SQLiteComicService struct {
	db *sql.DB
}

func NewSQLiteComicService(path string) (*SQLiteComicService, error) {
	if path == "" {
		path = findSQLiteComicDB()
	}
	if path == "" {
		return nil, fmt.Errorf("comic sqlite database not found")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteComicService{db: db}, nil
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

func (s *SQLiteComicService) Close() error {
	return s.db.Close()
}

func (s *SQLiteComicService) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLiteComicService) List(
	ctx context.Context,
	offset int,
	limit int,
	onlyTracked bool,
	onlyUnchecked bool,
	full bool,
) (ComicListResult, error) {
	where, args := comicFilters("", onlyTracked, onlyUnchecked)
	if full {
		return s.queryComics(ctx, where, args, 0, 0)
	}
	return s.queryComics(ctx, where, args, offset, limit)
}

func (s *SQLiteComicService) Search(
	ctx context.Context,
	title string,
	offset int,
	limit int,
	onlyTracked bool,
	onlyUnchecked bool,
	full bool,
) (ComicListResult, error) {
	where, args := comicFilters(title, onlyTracked, onlyUnchecked)
	if full {
		return s.queryComics(ctx, where, args, 0, 0)
	}
	return s.queryComics(ctx, where, args, offset, limit)
}

func comicFilters(title string, onlyTracked bool, onlyUnchecked bool) (string, []any) {
	filters := []string{"deleted = 0"}
	args := []any{}
	if title != "" {
		filters = append(filters, "LOWER(titles) LIKE LOWER(?)")
		args = append(args, "%"+title+"%")
	}
	if onlyTracked {
		filters = append(filters, "track = 1")
	}
	if onlyUnchecked {
		filters = append(filters, "track = 1", "current_chap != viewed_chap")
	}
	return "WHERE " + strings.Join(filters, " AND "), args
}

func (s *SQLiteComicService) queryComics(
	ctx context.Context,
	where string,
	args []any,
	offset int,
	limit int,
) (ComicListResult, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM comics "+where, args...).Scan(&total); err != nil {
		return ComicListResult{}, err
	}

	query := baseComicSelect() + " " + where + " ORDER BY last_update DESC, id"
	queryArgs := append([]any{}, args...)
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return ComicListResult{}, err
	}
	defer rows.Close()

	comics, err := scanComics(rows)
	if err != nil {
		return ComicListResult{}, err
	}

	totalPages := 1
	currentPage := 1
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
		currentPage = offset/limit + 1
	}
	return ComicListResult{
		Comics:      comics,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: currentPage,
	}, nil
}

func (s *SQLiteComicService) Get(ctx context.Context, id int) (ComicJSON, error) {
	row := s.db.QueryRowContext(ctx, baseComicSelect()+" WHERE id = ?", id)
	comic, err := scanComic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ComicJSON{}, ErrComicNotFound
	}
	return comic, err
}

func (s *SQLiteComicService) Create(ctx context.Context, comic ComicJSON) (ComicJSON, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(
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
		return ComicJSON{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return ComicJSON{}, err
	}
	return s.Get(ctx, int(id))
}

func (s *SQLiteComicService) Update(ctx context.Context, id int, patch ComicJSON) (ComicJSON, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return ComicJSON{}, err
	}

	if len(patch.Titles) > 0 {
		current.Titles = patch.Titles
	}
	if patch.Cover != "" && patch.Cover != current.Cover {
		current.Cover = patch.Cover
		current.CoverVisible = true
	}
	current.CurrentChap = valueOrCurrent(patch.CurrentChap, current.CurrentChap)
	current.ComType = valueOrCurrent(patch.ComType, current.ComType)
	current.Status = valueOrCurrent(patch.Status, current.Status)
	current.ViewedChap = valueOrCurrent(patch.ViewedChap, current.ViewedChap)
	current.Rating = valueOrCurrent(patch.Rating, current.Rating)
	if len(patch.PublishedIn) > 0 {
		current.PublishedIn = patch.PublishedIn
	}
	if len(patch.Genres) > 0 {
		current.Genres = patch.Genres
	}
	if patch.Description != "" {
		current.Description = patch.Description
	}
	if patch.Author != "" {
		current.Author = patch.Author
	}
	current.Track = patch.Track

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE comics SET titles = ?, current_chap = ?, cover = ?, last_update = ?,
			com_type = ?, status = ?, published_in = ?, genres = ?, description = ?,
			author = ?, track = ?, viewed_chap = ?, rating = ?, deleted = ?,
			cover_visible = ?
		WHERE id = ?`,
		strings.Join(current.Titles, "|"),
		current.CurrentChap,
		current.Cover,
		time.Now().Unix(),
		current.ComType,
		current.Status,
		joinInts(current.PublishedIn),
		joinInts(current.Genres),
		current.Description,
		current.Author,
		boolInt(current.Track),
		current.ViewedChap,
		current.Rating,
		boolInt(current.Deleted),
		current.CoverVisible,
		id,
	)
	if err != nil {
		return ComicJSON{}, err
	}
	return s.Get(ctx, id)
}

func (s *SQLiteComicService) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM comics WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrComicNotFound
	}
	return nil
}

func (s *SQLiteComicService) UpdateCoverVisibility(
	ctx context.Context,
	id int,
	cover string,
	visible bool,
) (ComicJSON, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return ComicJSON{}, err
	}
	if current.Cover != cover {
		return current, nil
	}
	_, err = s.db.ExecContext(
		ctx,
		"UPDATE comics SET cover_visible = ? WHERE id = ?",
		visible,
		id,
	)
	if err != nil {
		return ComicJSON{}, err
	}
	return s.Get(ctx, id)
}

func (s *SQLiteComicService) Merge(ctx context.Context, baseID int, mergingID int) (ComicJSON, error) {
	if baseID == mergingID {
		return ComicJSON{}, fmt.Errorf("Comics cannot merge with themselves")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ComicJSON{}, err
	}
	defer tx.Rollback() // nolint:errcheck

	base, err := scanComic(tx.QueryRowContext(ctx, baseComicSelect()+" WHERE id = ?", baseID))
	if errors.Is(err, sql.ErrNoRows) {
		return ComicJSON{}, ErrComicNotFound
	}
	if err != nil {
		return ComicJSON{}, err
	}

	duplicate, err := scanComic(tx.QueryRowContext(ctx, baseComicSelect()+" WHERE id = ?", mergingID))
	if errors.Is(err, sql.ErrNoRows) {
		return ComicJSON{}, ErrComicNotFound
	}
	if err != nil {
		return ComicJSON{}, err
	}
	if duplicate.ComType != 0 && base.ComType != duplicate.ComType {
		return ComicJSON{}, fmt.Errorf("Comics to merge should be of the same type")
	}

	merged := mergeComicValues(base, duplicate)
	_, err = tx.ExecContext(
		ctx,
		`UPDATE comics SET titles = ?, current_chap = ?, cover = ?, last_update = ?,
			com_type = ?, status = ?, published_in = ?, genres = ?, description = ?,
			author = ?, track = ?, viewed_chap = ?, rating = ?, cover_visible = ?
		WHERE id = ?`,
		strings.Join(merged.Titles, "|"),
		merged.CurrentChap,
		merged.Cover,
		time.Now().Unix(),
		merged.ComType,
		merged.Status,
		joinInts(merged.PublishedIn),
		joinInts(merged.Genres),
		merged.Description,
		merged.Author,
		boolInt(merged.Track),
		merged.ViewedChap,
		merged.Rating,
		merged.CoverVisible,
		baseID,
	)
	if err != nil {
		return ComicJSON{}, err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM comics WHERE id = ?", mergingID); err != nil {
		return ComicJSON{}, err
	}
	if err = tx.Commit(); err != nil {
		return ComicJSON{}, err
	}
	return s.Get(ctx, baseID)
}

func baseComicSelect() string {
	return `SELECT id, titles, current_chap, cover, CAST(last_update AS TEXT),
		com_type, status, published_in, genres, description, author, track,
		viewed_chap, rating, deleted, cover_visible FROM comics`
}

type comicScanner interface {
	Scan(dest ...any) error
}

func scanComics(rows *sql.Rows) ([]ComicJSON, error) {
	comics := []ComicJSON{}
	for rows.Next() {
		comic, err := scanComic(rows)
		if err != nil {
			return nil, err
		}
		comics = append(comics, comic)
	}
	return comics, rows.Err()
}

func scanComic(row comicScanner) (ComicJSON, error) {
	var titles, lastUpdate, publishedIn, genres string
	var comic ComicJSON
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
		return ComicJSON{}, err
	}
	comic.Titles = splitTitles(titles)
	comic.PublishedIn = splitInts(publishedIn)
	comic.Genres = splitInts(genres)
	comic.LastUpdate = formatLastUpdate(lastUpdate)
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
		return []int{0}
	}
	parts := strings.Split(value, "|")
	values := []int{}
	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			values = append(values, parsed)
		}
	}
	if len(values) == 0 {
		return []int{0}
	}
	return values
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return "0"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, "|")
}

func formatLastUpdate(value string) string {
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func valueOrCurrent(value int, current int) int {
	if value == 0 {
		return current
	}
	return value
}

func coverVisibleOrDefault(comic ComicJSON) bool {
	if comic.Cover == "" {
		return true
	}
	return comic.CoverVisible
}

func mergeComicValues(base ComicJSON, duplicate ComicJSON) ComicJSON {
	base.Titles = mergeStrings(base.Titles, duplicate.Titles)
	base.PublishedIn = mergeInts(base.PublishedIn, duplicate.PublishedIn)
	base.Genres = mergeInts(base.Genres, duplicate.Genres)
	base.CurrentChap = max(base.CurrentChap, duplicate.CurrentChap)
	base.ViewedChap = max(base.ViewedChap, duplicate.ViewedChap)
	base.Track = base.Track || duplicate.Track
	if base.Rating == 0 {
		base.Rating = duplicate.Rating
	}
	if base.Author == "" {
		base.Author = duplicate.Author
	}
	if base.Description == "" {
		base.Description = duplicate.Description
	}
	if base.Cover == "" || !base.CoverVisible {
		if duplicate.Cover != "" && duplicate.CoverVisible {
			base.Cover = duplicate.Cover
			base.CoverVisible = true
		}
	}
	return base
}

func mergeStrings(current []string, incoming []string) []string {
	seen := map[string]bool{}
	merged := []string{}
	for _, value := range append(current, incoming...) {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	return merged
}

func mergeInts(current []int, incoming []int) []int {
	seen := map[int]bool{}
	merged := []int{}
	for _, value := range append(current, incoming...) {
		if seen[value] {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	sort.Ints(merged)
	return merged
}
