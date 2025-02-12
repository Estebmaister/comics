-- name: GetComicById :one
SELECT * FROM comics
WHERE id = $1 LIMIT 1;

-- name: GetComicsByTitle :many
SELECT * FROM comics
WHERE EXISTS (
    SELECT 1
    FROM unnest(titles) AS title
    WHERE title LIKE '%' || $1 || '%'
)
ORDER BY last_update DESC
LIMIT $2 OFFSET $3;

-- name: GetComics :many
SELECT * FROM comics
ORDER BY last_update DESC
LIMIT $1 OFFSET $2;

-- name: CreateComic :one
INSERT INTO comics (
  titles, author, description, cover, 
  com_type, status, published_in, genres, 
  current_chap, viewed_chap, last_update, track
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateComicById :exec
UPDATE comics
  set titles = $2,
  author = $3,
  description = $4,
  cover = $5,
  com_type = $6,
  status = $7,
  published_in = $8,
  genres = $9,
  current_chap = $10,
  viewed_chap = $11,
  last_update = $12,
  track = $13,
  deleted = $14
WHERE id = $1;

-- name: SoftDeleteComicById :exec
UPDATE comics
SET deleted = true
WHERE id = $1;

-- name: DeleteComicById :exec
DELETE FROM comics
WHERE id = $1;