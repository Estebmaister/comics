-- name: GetComicByID :one
SELECT * FROM comics
WHERE id = ? LIMIT 1;

-- name: GetComicsByTitle :many
SELECT * FROM comics
WHERE titles LIKE '%' || ? || '%'
ORDER BY last_update DESC
LIMIT ? OFFSET ?;

-- name: GetComics :many
SELECT * FROM comics
ORDER BY last_update DESC
LIMIT ? OFFSET ?;

-- name: CreateComic :one
INSERT INTO comics (
  titles, author, description, cover, 
  com_type, status, published_in, genres, 
  current_chap, viewed_chap, last_update, track
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: UpdateComicByID :exec
UPDATE comics
  set titles = ?,
  author = ?,
  description = ?,
  cover = ?,
  com_type = ?,
  status = ?,
  rating = ?,
  published_in = ?,
  genres = ?,
  current_chap = ?,
  viewed_chap = ?,
  last_update = ?,
  track = ?,
  deleted = ?
WHERE id = ?;

-- name: SoftDeleteComicByID :exec
UPDATE comics
SET deleted = true, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteComicByID :exec
DELETE FROM comics
WHERE id = ?;