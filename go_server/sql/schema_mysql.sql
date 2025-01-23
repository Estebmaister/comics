CREATE TABLE IF NOT EXISTS comics (
  id INT AUTO_INCREMENT PRIMARY KEY,
  author VARCHAR(150),
  description VARCHAR(2000),
  cover VARCHAR(2083),
  com_type INT NOT NULL,
  status INT NOT NULL,
  current_chap INT NOT NULL,
  viewed_chap INT NOT NULL,
  last_update TIMESTAMP NOT NULL,
  track BOOLEAN NOT NULL,
  deleted BOOLEAN NOT NULL
);

CREATE TABLE comic_titles (
    comic_id INT NOT NULL,
    title VARCHAR(250) NOT NULL,
    FOREIGN KEY (comic_id) REFERENCES comics(id)
);

CREATE TABLE comic_publishers (
    comic_id INT NOT NULL,
    publisher INT NOT NULL,
    FOREIGN KEY (comic_id) REFERENCES comics(id)
);

CREATE TABLE comic_genres (
    comic_id INT NOT NULL,
    genre INT NOT NULL,
    FOREIGN KEY (comic_id) REFERENCES comics(id)
);