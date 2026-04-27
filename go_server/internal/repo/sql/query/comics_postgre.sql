-- name: GetComicByID :one
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
  titles, author, description, cover, cover_visible,
  com_type, status, published_in, genres, 
  current_chap, viewed_chap, last_update, track
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: UpdateComicByID :exec
UPDATE comics
  set titles = $2,
  author = $3,
  description = $4,
  cover = $5,
  cover_visible = $6,
  com_type = $7,
  status = $8,
  published_in = $9,
  genres = $10,
  current_chap = $11,
  viewed_chap = $12,
  last_update = $13,
  track = $14,
  deleted = $15
WHERE id = $1;

-- name: SoftDeleteComicByID :exec
UPDATE comics
SET deleted = true
WHERE id = $1;

-- name: DeleteComicByID :exec
DELETE FROM comics
WHERE id = $1;
