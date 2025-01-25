CREATE TABLE comics (
  id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  titles        TEXT    NOT NULL,
  author        TEXT,
  description   TEXT,
  cover         TEXT,
  published_in  TEXT    NOT NULL DEFAULT '0',
  genres        TEXT    NOT NULL DEFAULT '0',
  com_type      INTEGER NOT NULL DEFAULT 0,
  status        INTEGER NOT NULL DEFAULT 0,
  rating        INTEGER NOT NULL DEFAULT 0,
  current_chap  INTEGER NOT NULL DEFAULT 0,
  viewed_chap   INTEGER NOT NULL DEFAULT 0,
  track         BOOLEAN NOT NULL DEFAULT 0,
  deleted       BOOLEAN NOT NULL DEFAULT 0,
  last_update   DATE    NOT NULL DEFAULT CURRENT_TIMESTAMP
);