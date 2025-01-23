CREATE TABLE comics (
  id INTEGER PRIMARY KEY,
  titles TEXT NOT NULL,
  author TEXT,
  description TEXT,
  cover TEXT,
  com_type INTEGER NOT NULL,
  status INTEGER NOT NULL,
  published_in TEXT,
  genres TEXT NOT NULL,
  current_chap INTEGER NOT NULL,
  viewed_chap INTEGER NOT NULL,
  last_update INTEGER NOT NULL,
  track BOOLEAN NOT NULL,
  deleted BOOLEAN NOT NULL
);