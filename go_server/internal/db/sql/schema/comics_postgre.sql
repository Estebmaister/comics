CREATE TABLE IF NOT EXISTS comics (
  id            SERIAL          PRIMARY KEY,
  titles        VARCHAR(255)[]  NOT NULL,
  author        VARCHAR(255)    NULL,
  description   VARCHAR(2083)   NULL,
  cover         VARCHAR(2083)   NULL,
  published_in  INTEGER[]       NOT NULL,
  genres        INTEGER[]       NOT NULL,
  com_type      INTEGER         NOT NULL DEFAULT 0,
  status        INTEGER         NOT NULL DEFAULT 0,
  rating        INTEGER         NOT NULL DEFAULT 0,
  current_chap  INTEGER         NOT NULL DEFAULT 0,
  viewed_chap   INTEGER         NOT NULL DEFAULT 0,
  track         BOOLEAN         NOT NULL DEFAULT false,
  deleted       BOOLEAN         NOT NULL DEFAULT false,
  last_update   DATE            NOT NULL
);