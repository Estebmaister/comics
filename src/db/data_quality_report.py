import json
from collections import Counter, defaultdict
from typing import Dict, List, Tuple

from sqlalchemy.sql import text

from db import ComicDB, Session
from helpers.text import normalize_text


def _normalize_titles(titles: List[str]) -> List[str]:
    return [normalize_text(title).capitalize() for title in titles]


def _safe_pct(part: int, total: int) -> float:
    return round((part / total) * 100, 2) if total else 0.0


def _title_key(titles: List[str]) -> str:
    if not titles:
        return ""
    normalized = _normalize_titles(titles)
    return normalized[0].lower()


def run() -> Dict[str, object]:
    with Session() as session:
        total = session.query(ComicDB).count()
        comics = session.query(ComicDB).all()

    missing_author = 0
    missing_description = 0
    missing_titles = 0
    viewed_gt_current = 0
    negative_chapters = 0
    title_normalization_changes = 0

    title_counts: Counter[str] = Counter()
    duplicate_title_groups: Dict[str, List[int]] = defaultdict(list)

    for comic in comics:
        titles = comic.get_titles()
        if not titles or any(title.strip() == "" for title in titles):
            missing_titles += 1

        if not comic.author or str(comic.author).strip() == "":
            missing_author += 1
        if not comic.description or str(comic.description).strip() == "":
            missing_description += 1

        if comic.viewed_chap > comic.current_chap:
            viewed_gt_current += 1
        if comic.viewed_chap < 0 or comic.current_chap < 0:
            negative_chapters += 1

        normalized = _normalize_titles(titles)
        if normalized != titles:
            title_normalization_changes += 1

        key = _title_key(titles)
        if key:
            title_counts[key] += 1
            duplicate_title_groups[key].append(comic.id)

    duplicate_titles = {
        title: ids
        for title, ids in duplicate_title_groups.items()
        if len(ids) > 1
    }

    report = {
        "total_comics": total,
        "missing_author": {
            "count": missing_author,
            "percent": _safe_pct(missing_author, total),
        },
        "missing_description": {
            "count": missing_description,
            "percent": _safe_pct(missing_description, total),
        },
        "missing_titles": {
            "count": missing_titles,
            "percent": _safe_pct(missing_titles, total),
        },
        "invalid_chapters": {
            "viewed_greater_than_current": viewed_gt_current,
            "negative_values": negative_chapters,
        },
        "title_normalization_changes": title_normalization_changes,
        "duplicate_title_groups": duplicate_titles,
    }
    return report


if __name__ == "__main__":
    result = run()
    print(json.dumps(result, indent=2))
