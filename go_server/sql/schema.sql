CREATE TABLE IF NOT EXISTS comics (
  id SERIAL PRIMARY KEY,
  titles VARCHAR(250)[] NOT NULL,
  author VARCHAR(150),
  description VARCHAR(2000),
  cover VARCHAR(2083),
  com_type INTEGER NOT NULL,
  status INTEGER NOT NULL,
  published_in INTEGER[],
  genres INTEGER[] NOT NULL,
  current_chap INTEGER NOT NULL,
  viewed_chap INTEGER NOT NULL,
  last_update INTEGER NOT NULL,
  track BOOLEAN NOT NULL,
  deleted BOOLEAN NOT NULL
);