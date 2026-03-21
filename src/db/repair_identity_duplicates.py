#!/usr/bin/env python3

from __future__ import annotations

import argparse
import shutil
import sys
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Iterable, List

from sqlalchemy.sql import text

ROOT = Path(__file__).resolve().parents[2]
SRC = ROOT / "src"

for path in (ROOT, SRC):
    path_str = str(path)
    if path_str not in sys.path:
        sys.path.insert(0, path_str)

from db import ComicDB, Session, Types, comic_file, db_file
from db.identity import merge_unique_values, normalize_title_variants
from db.repo import rebuild_json_backup_from_db


@dataclass
class DuplicateGroup:
    identity_key: str
    comics: List[ComicDB]

    @property
    def typed_variants(self) -> set[int]:
        return {
            int(comic.com_type)
            for comic in self.comics
            if int(comic.com_type) != int(Types.Unknown)
        }

    @property
    def ambiguous(self) -> bool:
        non_novel_types = {
            comic_type
            for comic_type in self.typed_variants
            if comic_type != int(Types.Novel)
        }
        return len(non_novel_types) > 1


def _merge_unique_strings(current_values: Iterable[str], incoming_values: Iterable[str]) -> List[str]:
    merged: List[str] = []
    for value in list(current_values) + list(incoming_values):
        clean_value = str(value).strip()
        if clean_value and clean_value not in merged:
            merged.append(clean_value)
    return merged


def _prefer_text(current_value: str, incoming_value: str) -> str:
    if not current_value and incoming_value:
        return incoming_value
    if incoming_value and len(incoming_value) > len(current_value):
        return incoming_value
    return current_value


def _backup_storage() -> Path:
    recovery_dir = Path("src/db/recovery")
    recovery_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
    backup_dir = recovery_dir / f"identity-repair-{timestamp}"
    backup_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(db_file, backup_dir / Path(db_file).name)
    shutil.copy2(comic_file, backup_dir / Path(comic_file).name)
    return backup_dir


def _load_duplicate_groups(session) -> List[DuplicateGroup]:
    grouped: Dict[str, List[ComicDB]] = defaultdict(list)
    comics = session.query(ComicDB).filter(ComicDB.deleted == 0).order_by(ComicDB.id).all()
    for comic in comics:
        if comic.identity_key:
            grouped[comic.identity_key].append(comic)

    return [
        DuplicateGroup(identity_key=identity_key, comics=matches)
        for identity_key, matches in grouped.items()
        if len(matches) > 1
    ]


def _merge_group(session, group: DuplicateGroup) -> int:
    canonical = sorted(group.comics, key=lambda comic: comic.id)[0]
    merged_count = 0

    for duplicate in sorted(group.comics, key=lambda comic: comic.id)[1:]:
        canonical.set_titles(
            _merge_unique_strings(canonical.get_titles(), duplicate.get_titles())
        )
        canonical.set_genres(
            merge_unique_values(canonical.get_genres(), duplicate.get_genres())
        )
        canonical.set_published_in(
            merge_unique_values(
                canonical.get_published_in(),
                duplicate.get_published_in(),
            )
        )
        canonical.current_chap = max(canonical.current_chap, duplicate.current_chap)
        canonical.viewed_chap = max(canonical.viewed_chap, duplicate.viewed_chap)
        canonical.track = int(bool(canonical.track or duplicate.track))
        canonical.rating = max(canonical.rating, duplicate.rating)
        canonical.last_update = max(canonical.last_update, duplicate.last_update)

        if canonical.com_type == int(Types.Unknown) and duplicate.com_type != int(Types.Unknown):
            canonical.com_type = duplicate.com_type
        if canonical.status == int(Types.Unknown) and duplicate.status != int(Types.Unknown):
            canonical.status = duplicate.status

        canonical.author = _prefer_text(canonical.author, duplicate.author)
        canonical.description = _prefer_text(canonical.description, duplicate.description)
        canonical.cover = _prefer_text(canonical.cover, duplicate.cover)

        session.delete(duplicate)
        merged_count += 1

    canonical.normalize_titles()
    return merged_count


def _ensure_identity_unique_index(session) -> None:
    session.execute(
        text(
            """
            CREATE UNIQUE INDEX IF NOT EXISTS uq_comics_identity_key
            ON comics (identity_key)
            WHERE deleted = 0
            """
        )
    )


def _load_catalog_title_normalizations(session) -> List[tuple[ComicDB, List[str]]]:
    changes: List[tuple[ComicDB, List[str]]] = []
    comics = session.query(ComicDB).filter(ComicDB.deleted == 0).order_by(ComicDB.id).all()
    for comic in comics:
        normalized_titles = normalize_title_variants(
            comic.get_titles(),
            comic.com_type,
        )
        normalized_storage = "|".join(normalized_titles)
        if str(comic.titles or "") != normalized_storage:
            changes.append((comic, normalized_titles))
    return changes


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Audit and repair duplicate comics grouped by identity_key."
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Apply merges for clear duplicate groups and rebuild the JSON backup.",
    )
    parser.add_argument(
        "--merge-ambiguous",
        action="store_true",
        help=(
            "Merge cross-type duplicate groups too. The lowest comic id wins and "
            "keeps its type when there is a conflict."
        ),
    )
    parser.add_argument(
        "--normalize-all-titles",
        action="store_true",
        help=(
            "Normalize the full catalog title list to sentence-case storage and "
            "smart-quote cleanup, even for rows that are already unique."
        ),
    )
    args = parser.parse_args()

    with Session() as session:
        groups = _load_duplicate_groups(session)
        clear_groups = [group for group in groups if not group.ambiguous]
        ambiguous_groups = [group for group in groups if group.ambiguous]
        merge_groups = clear_groups + ambiguous_groups if args.merge_ambiguous else clear_groups
        title_normalizations = _load_catalog_title_normalizations(session)

        print(f"Duplicate identity groups: {len(groups)}")
        print(f"Clear merge groups: {len(clear_groups)}")
        print(f"Ambiguous review groups: {len(ambiguous_groups)}")
        print(f"Merge target groups: {len(merge_groups)}")
        print(f"Catalog title normalizations: {len(title_normalizations)}")

        for group in ambiguous_groups:
            titles = [comic.get_titles()[0] for comic in group.comics]
            print(
                f"AMBIGUOUS {group.identity_key}: "
                f"types={[int(comic.com_type) for comic in group.comics]} "
                f"titles={titles}"
            )

        if not args.apply:
            return 0

        backup_dir = _backup_storage()
        merged_rows = 0
        for group in merge_groups:
            merged_rows += _merge_group(session, group)
        normalized_rows = 0
        if args.normalize_all_titles:
            for comic, normalized_titles in title_normalizations:
                comic.set_titles(normalized_titles)
                normalized_rows += 1

        session.commit()
        rebuild_json_backup_from_db()

        remaining_groups = _load_duplicate_groups(session)
        if not remaining_groups:
            _ensure_identity_unique_index(session)
            session.commit()

        print(f"Backup written to: {backup_dir}")
        print(f"Merged rows: {merged_rows}")
        print(f"Normalized rows: {normalized_rows}")
        print(f"Remaining duplicate groups: {len(remaining_groups)}")
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
