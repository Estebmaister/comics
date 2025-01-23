-- name: GetComicById :one
SELECT c.*, 
  GROUP_CONCAT(ct.title ORDER BY ct.title) AS titles,
  GROUP_CONCAT(cp.publisher ORDER BY cp.publisher) AS published_in,
  GROUP_CONCAT(cg.genre ORDER BY cg.genre) AS genres
FROM comics c
  JOIN comic_titles ct ON c.id = ct.comic_id
  JOIN comic_genres cg ON c.id = cg.comic_id
  JOIN comic_publishers cp ON c.id = cp.comic_id
WHERE c.id = ?
GROUP BY c.id LIMIT 1;

-- name: GetComicsByTitle :many
-- Find comics with titles matching a substring
SELECT c.*, 
  GROUP_CONCAT(ct.title ORDER BY ct.title) AS titles,
  GROUP_CONCAT(cp.publisher ORDER BY cp.publisher) AS published_in,
  GROUP_CONCAT(cg.genre ORDER BY cg.genre) AS genres
FROM comics c
  JOIN comic_titles ct ON c.id = ct.comic_id
  JOIN comic_genres cg ON c.id = cg.comic_id
  JOIN comic_publishers cp ON c.id = cp.comic_id
WHERE ct.title LIKE CONCAT('%', ?, '%')
GROUP BY c.id
ORDER BY c.last_update DESC
LIMIT ? OFFSET ?;

-- name: GetComics :many
SELECT c.*, 
  GROUP_CONCAT(ct.title ORDER BY ct.title) AS titles,
  GROUP_CONCAT(cp.publisher ORDER BY cp.publisher) AS published_in,
  GROUP_CONCAT(cg.genre ORDER BY cg.genre) AS genres
FROM comics c
  JOIN comic_titles ct ON c.id = ct.comic_id
  JOIN comic_genres cg ON c.id = cg.comic_id
  JOIN comic_publishers cp ON c.id = cp.comic_id
GROUP BY c.id
ORDER BY c.last_update DESC
LIMIT ? OFFSET ?;

-- name: GetLastInsertID :one
SELECT LAST_INSERT_ID() AS id;

-- CreateComic start
-- name: InsertComic :exec
INSERT INTO comics (
  author, description, cover, com_type, status, current_chap, 
  viewed_chap, last_update, track, deleted )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
-- name: InsertTitle :exec
INSERT INTO comic_titles (comic_id, title) VALUES (?, ?);
-- name: InsertPublisher :exec
INSERT INTO comic_publishers (comic_id, publisher) VALUES (?, ?);
-- name: InsertGenre :exec
INSERT INTO comic_genres (comic_id, genre) VALUES (?, ?); -- CreateComic end


-- UpdateComic start
-- name: UpdateComicById :exec
UPDATE comics 
  set author = ?,
  description = ?,
  cover = ?,
  com_type = ?,
  status = ?,
  current_chap = ?,
  viewed_chap = ?,
  last_update = ?,
  track = ?,
  deleted = ?
WHERE id = ?; -- delete and re-insert all title, publisher, genre if needed, UpdateComic end


-- name: SoftDeleteComicById :exec
UPDATE comics SET deleted = true WHERE id = ?;


-- DeleteComic start
-- name: HardDeleteComicById :exec
DELETE FROM comics            WHERE id = ?;
-- name: DeleteTitlesByComicID :exec
DELETE FROM comic_titles      WHERE comic_id = ?;
-- name: DeletePublishersByComicID :exec
DELETE FROM comic_publishers  WHERE comic_id = ?;
-- name: DeleteGenresByComicID :exec
DELETE FROM comic_genres      WHERE comic_id = ?; -- DeleteComic end