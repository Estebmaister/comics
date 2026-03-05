import json
from typing import List

from sqlalchemy.sql import text

from db import ComicDB, Session, load_comics, save_comics_file
from helpers.text import normalize_text


def normalize_titles(titles: List[str]) -> List[str]:
    return [normalize_text(title).capitalize() for title in titles]


def run() -> int:
    updated = 0
    with Session() as session:
        comics = session.query(ComicDB).all()
        for comic in comics:
            normalized = normalize_titles(comic.get_titles())
            if normalized != comic.get_titles():
                comic.set_titles(normalized)
                updated += 1
        session.commit()

    if load_comics:
        for comic in load_comics:
            normalized = normalize_titles(comic.get('titles', []))
            comic['titles'] = normalized
        save_comics_file(load_comics)

    return updated


if __name__ == "__main__":
    count = run()
    print(f"Normalized titles for {count} comics.")
