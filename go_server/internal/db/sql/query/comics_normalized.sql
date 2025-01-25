-- name: GetComicByID :one
-- Distinct is used to avoid Cartesian product on Joins
-- TODO: implement replacement for default separator ',' 
--       e.g. REPLACE(ct.title, ',', '|||')
SELECT c.*, 
  GROUP_CONCAT(DISTINCT ct.title) AS titles,
  GROUP_CONCAT(DISTINCT cp.publisher) AS published_in,
  GROUP_CONCAT(DISTINCT cg.genre) AS genres
FROM comics c
  JOIN titles ct ON c.id = ct.comic_id
  LEFT JOIN genres cg ON c.id = cg.comic_id
  LEFT JOIN publishers cp ON c.id = cp.comic_id
WHERE c.id = ?
GROUP BY c.id LIMIT 1;

-- name: GetComicsByTitle :many
-- Find comics with titles matching a substring
SELECT c.*, 
  GROUP_CONCAT(DISTINCT ct.title) AS titles,
  GROUP_CONCAT(DISTINCT cp.publisher) AS published_in,
  GROUP_CONCAT(DISTINCT cg.genre) AS genres
FROM comics c
  JOIN titles ct ON c.id = ct.comic_id
  LEFT JOIN genres cg ON c.id = cg.comic_id
  LEFT JOIN publishers cp ON c.id = cp.comic_id
WHERE ct.title LIKE '%' || ? || '%'
GROUP BY c.id
ORDER BY c.last_update DESC
LIMIT ? OFFSET ?;

-- name: GetComics :many
-- Find all comics ordered by last_update with pagination
SELECT c.*, 
  GROUP_CONCAT(DISTINCT ct.title) AS titles,
  GROUP_CONCAT(DISTINCT cp.publisher) AS published_in,
  GROUP_CONCAT(DISTINCT cg.genre) AS genres
FROM comics c
  JOIN titles ct ON c.id = ct.comic_id
  LEFT JOIN genres cg ON c.id = cg.comic_id
  LEFT JOIN publishers cp ON c.id = cp.comic_id
GROUP BY c.id
ORDER BY c.last_update DESC
LIMIT ? OFFSET ?;

-- CreateComic start
-- name: InsertComic :one
-- TODO: implement transaction to create comics
INSERT INTO comics (
  author, description, cover, com_type, status, 
  current_chap, viewed_chap, track, deleted, last_update )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;
-- name: InsertTitle :exec
INSERT INTO titles      (comic_id, title)     VALUES (?, ?);
-- name: InsertPublisher :exec
INSERT INTO publishers  (comic_id, publisher) VALUES (?, ?);
-- name: InsertGenre :exec
INSERT INTO genres      (comic_id, genre)     VALUES (?, ?); -- CreateComic end


-- UpdateComic start
-- name: UpdateComicByID :exec
-- TODO: implement transaction to update comics
UPDATE comics 
  set author = ?, description = ?, cover = ?, com_type = ?, status = ?,
  current_chap = ?, viewed_chap = ?, track = ?, deleted = ?, last_update = ?
WHERE id = ?; -- delete and re-insert all title, publisher, genre if needed, UpdateComic end


-- name: SoftDeleteComicByID :exec
UPDATE comics SET deleted = true WHERE id = ?;


-- DeleteComic start
-- name: HardDeleteComicByID :exec
-- TODO: implement transaction to delete comics
DELETE FROM comics      WHERE id = ?;
-- name: DeleteTitlesByComicID :exec
DELETE FROM titles      WHERE comic_id = ?;
-- name: DeletePublishersByComicID :exec
DELETE FROM publishers  WHERE comic_id = ?;
-- name: DeleteGenresByComicID :exec
DELETE FROM genres      WHERE comic_id = ?; -- DeleteComic end