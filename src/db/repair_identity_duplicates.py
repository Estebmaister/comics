#!/usr/bin/env python3

from __future__ import annotations

import argparse
import csv
import json
import shutil
import sys
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Set, Tuple

from sqlalchemy.sql import text

ROOT = Path(__file__).resolve().parents[2]
SRC = ROOT / "src"

for path in (SRC,):
    path_str = str(path)
    if path_str not in sys.path:
        sys.path.insert(0, path_str)

from db import ComicDB, Session, Statuses, Types, comic_file, db_file
from db.identity import (
    is_novel_identity,
    merge_unique_values,
    normalize_title_variants,
    title_match_key,
    titles_are_prefix_match,
)
from db.repo import rebuild_json_backup_from_db

DEFAULT_MIN_KEY_RATIO = 0.72
DEFAULT_MAX_GROUP_SIZE = 4
DEFAULT_CHAPTER_SPREAD_MAX = 15
DEFAULT_CHAPTER_RATIO_MIN = 0.85
CHAPTER_FAR_SPREAD = 50
CHAPTER_FAR_RATIO = 0.5
MINIMUM_KEY_LENGTH = 18


@dataclass
class DuplicateGroup:
    identity_key: str
    comics: List[ComicDB]
    reason: str = "identity_key"

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

    @property
    def cross_type(self) -> bool:
        non_novel = {
            int(comic.com_type)
            for comic in self.comics
            if int(comic.com_type) not in (int(Types.Unknown), int(Types.Novel))
        }
        return len(non_novel) > 1


def _merge_unique_strings(
    current_values: Iterable[str], incoming_values: Iterable[str]
) -> List[str]:
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


def _prefer_cover(canonical: ComicDB, duplicate: ComicDB) -> tuple[str, bool]:
    canonical_visible = bool(getattr(canonical, "cover_visible", True))
    duplicate_visible = bool(getattr(duplicate, "cover_visible", True))
    if canonical.cover and canonical_visible:
        return canonical.cover, canonical_visible
    if duplicate.cover and duplicate_visible:
        return duplicate.cover, duplicate_visible
    if canonical.cover:
        return canonical.cover, canonical_visible
    if duplicate.cover:
        return duplicate.cover, duplicate_visible
    return "", True


def _backup_storage() -> Path:
    recovery_dir = Path("src/db/recovery")
    recovery_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
    backup_dir = recovery_dir / f"identity-repair-{timestamp}"
    backup_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(db_file, backup_dir / Path(db_file).name)
    shutil.copy2(comic_file, backup_dir / Path(comic_file).name)
    return backup_dir


def _group_matches_title_like(group: DuplicateGroup, title_like: str) -> bool:
    if not title_like:
        return True
    needle = title_like.lower()
    return any(
        needle in title.lower()
        for comic in group.comics
        for title in comic.get_titles()
    )


def _is_non_novel_comic(comic: ComicDB) -> bool:
    if int(comic.com_type) == int(Types.Novel):
        return False
    return not any(
        is_novel_identity(title, comic.com_type) for title in comic.get_titles()
    )


def _dedupe_comics_by_id(comics: Iterable[ComicDB]) -> List[ComicDB]:
    seen: Dict[int, ComicDB] = {}
    for comic in comics:
        seen[int(comic.id)] = comic
    return sorted(seen.values(), key=lambda row: int(row.id))


def _comic_title_keys(comic: ComicDB, minimum_length: int = 0) -> List[str]:
    keys = [
        key
        for key in (title_match_key(title) for title in comic.get_titles())
        if len(key) >= minimum_length
    ]
    return keys or [
        key
        for key in (title_match_key(title) for title in comic.get_titles())
        if key
    ]


def _keys_match(
    left_key: str,
    right_key: str,
    min_key_ratio: float,
) -> bool:
    if not left_key or not right_key:
        return False
    if left_key == right_key:
        return True
    if titles_are_prefix_match(left_key, right_key) or titles_are_prefix_match(
        right_key, left_key
    ):
        shorter = min(len(left_key), len(right_key))
        longer = max(len(left_key), len(right_key))
        if longer == 0:
            return False
        return (shorter / longer) >= min_key_ratio
    return False


def _comics_share_title_match(
    left: ComicDB,
    right: ComicDB,
    min_key_ratio: float,
    minimum_key_length: int = MINIMUM_KEY_LENGTH,
) -> bool:
    left_keys = _comic_title_keys(left, minimum_key_length)
    right_keys = _comic_title_keys(right, minimum_key_length)
    if not left_keys or not right_keys:
        left_keys = _comic_title_keys(left, 0)
        right_keys = _comic_title_keys(right, 0)
    return any(
        _keys_match(source_key, target_key, min_key_ratio)
        for source_key in left_keys
        for target_key in right_keys
    )


def _connected_comic_ids(
    comics: List[ComicDB],
    min_key_ratio: float,
) -> Set[int]:
    if len(comics) < 2:
        return set()

    parent = {int(comic.id): int(comic.id) for comic in comics}

    def find(node: int) -> int:
        while parent[node] != node:
            parent[node] = parent[parent[node]]
            node = parent[node]
        return node

    def union(left: int, right: int) -> None:
        root_left = find(left)
        root_right = find(right)
        if root_left != root_right:
            parent[root_right] = root_left

    for index, comic in enumerate(comics):
        for other in comics[index + 1 :]:
            if _comics_share_title_match(comic, other, min_key_ratio):
                union(int(comic.id), int(other.id))

    groups: Dict[int, Set[int]] = defaultdict(set)
    for comic_id in parent:
        groups[find(comic_id)].add(comic_id)
    return max(groups.values(), key=len) if groups else set()


def _load_duplicate_groups(session, title_like: str = "") -> List[DuplicateGroup]:
    grouped: Dict[str, List[ComicDB]] = defaultdict(list)
    comics = session.query(ComicDB).filter(ComicDB.deleted == 0).order_by(ComicDB.id).all()
    for comic in comics:
        if comic.identity_key:
            grouped[comic.identity_key].append(comic)

    groups = [
        DuplicateGroup(identity_key=identity_key, comics=_dedupe_comics_by_id(matches))
        for identity_key, matches in grouped.items()
        if len(matches) > 1
    ]
    return [group for group in groups if _group_matches_title_like(group, title_like)]


def _load_title_prefix_duplicate_groups(
    session,
    title_like: str = "",
    min_key_ratio: float = DEFAULT_MIN_KEY_RATIO,
) -> List[DuplicateGroup]:
    query = session.query(ComicDB).filter(ComicDB.deleted == 0).order_by(ComicDB.id)
    if title_like:
        query = query.filter(ComicDB.titles.ilike(f"%{title_like}%"))
    comics = query.all()
    bucketed: Dict[tuple[int, str], Dict[int, ComicDB]] = defaultdict(dict)

    for comic in comics:
        keys = [
            key
            for key in (title_match_key(title) for title in comic.get_titles())
            if len(key) >= MINIMUM_KEY_LENGTH
        ]
        if not keys:
            continue
        for key in keys:
            bucketed[(int(comic.com_type), key[:MINIMUM_KEY_LENGTH])][int(comic.id)] = comic

    groups: List[DuplicateGroup] = []
    seen_group_keys: set[tuple[int, ...]] = set()
    for (comic_type, bucket), match_map in bucketed.items():
        matches = list(match_map.values())
        if len(matches) < 2:
            continue

        grouped_ids = _connected_comic_ids(matches, min_key_ratio)
        if len(grouped_ids) < 2:
            continue

        group_key = tuple(sorted(grouped_ids))
        if group_key in seen_group_keys:
            continue
        seen_group_keys.add(group_key)
        groups.append(
            DuplicateGroup(
                identity_key=f"title-prefix:{comic_type}:{bucket}",
                comics=_dedupe_comics_by_id(
                    [comic for comic in matches if int(comic.id) in grouped_ids]
                ),
                reason="title_prefix",
            )
        )

    return groups


def _load_cross_type_duplicate_groups(
    session,
    title_like: str = "",
    min_key_ratio: float = DEFAULT_MIN_KEY_RATIO,
) -> List[DuplicateGroup]:
    query = session.query(ComicDB).filter(ComicDB.deleted == 0).order_by(ComicDB.id)
    if title_like:
        query = query.filter(ComicDB.titles.ilike(f"%{title_like}%"))
    comics = [comic for comic in query.all() if _is_non_novel_comic(comic)]
    bucketed: Dict[str, Dict[int, ComicDB]] = defaultdict(dict)

    for comic in comics:
        keys = [
            key
            for key in (title_match_key(title) for title in comic.get_titles())
            if len(key) >= MINIMUM_KEY_LENGTH
        ]
        if not keys:
            continue
        for key in keys:
            bucketed[key[:MINIMUM_KEY_LENGTH]][int(comic.id)] = comic

    groups: List[DuplicateGroup] = []
    seen_group_keys: set[tuple[int, ...]] = set()
    for bucket, match_map in bucketed.items():
        matches = list(match_map.values())
        if len(matches) < 2:
            continue

        grouped_ids = _connected_comic_ids(matches, min_key_ratio)
        if len(grouped_ids) < 2:
            continue

        selected = _dedupe_comics_by_id(
            [comic for comic in matches if int(comic.id) in grouped_ids]
        )
        types = {
            int(comic.com_type)
            for comic in selected
            if int(comic.com_type) not in (int(Types.Unknown), int(Types.Novel))
        }
        if len(types) < 2:
            continue

        group_key = tuple(sorted(grouped_ids))
        if group_key in seen_group_keys:
            continue
        seen_group_keys.add(group_key)
        groups.append(
            DuplicateGroup(
                identity_key=f"cross-type:{bucket}",
                comics=selected,
                reason="cross_type",
            )
        )

    return groups


def _chapter_metrics(comics: List[ComicDB]) -> tuple[List[int], int, float]:
    chapters = [int(comic.current_chap or 0) for comic in comics]
    max_chap = max(chapters) if chapters else 0
    min_chap = min(chapters) if chapters else 0
    spread = max_chap - min_chap
    ratio = (min_chap / max_chap) if max_chap > 0 else 1.0
    return chapters, spread, ratio


def _has_exact_title_match(comics: List[ComicDB]) -> bool:
    keys: Set[str] = set()
    for comic in comics:
        for title in comic.get_titles():
            key = title_match_key(title)
            if key:
                keys.add(key)
    if len(keys) < 2:
        return len(comics) >= 2 and len(keys) == 1
    for left in keys:
        for right in keys:
            if left == right:
                return True
    return False


def _compute_signals(
    group: DuplicateGroup,
    min_key_ratio: float,
    chapter_spread_max: int,
    chapter_ratio_min: float,
) -> Dict[str, bool]:
    chapters, spread, ratio = _chapter_metrics(group.comics)
    max_chap = max(chapters) if chapters else 0
    chapters_close = spread <= chapter_spread_max or (
        max_chap >= 20 and ratio >= chapter_ratio_min
    )
    chapters_far = spread > CHAPTER_FAR_SPREAD or (
        max_chap >= 30 and ratio < CHAPTER_FAR_RATIO
    )
    title_prefix = group.reason in ("title_prefix", "cross_type", "identity_key_ambiguous_cross_type")
    exact_title = _has_exact_title_match(group.comics)
    return {
        "title_prefix": title_prefix,
        "exact_title": exact_title,
        "chapters_close": chapters_close,
        "chapters_far": chapters_far,
    }


def _assess_risk(
    group: DuplicateGroup,
    signals: Dict[str, bool],
    max_group_size: int,
    min_key_ratio: float,
) -> str:
    if len(group.comics) > max_group_size:
        return "review"
    if signals["chapters_far"]:
        return "review"
    if group.cross_type or group.reason == "cross_type":
        if signals["exact_title"] or (signals["title_prefix"] and signals["chapters_close"]):
            return "low"
        return "review"
    if signals["chapters_close"] and signals["title_prefix"]:
        return "low"
    if group.reason == "identity_key" and not group.ambiguous:
        return "low"
    return "review"


def _group_plan_entry(
    group: DuplicateGroup,
    min_key_ratio: float,
    max_group_size: int,
    chapter_spread_max: int,
    chapter_ratio_min: float,
) -> Dict[str, Any]:
    comics = _dedupe_comics_by_id(group.comics)
    canonical = sorted(comics, key=lambda comic: int(comic.id))[0]
    merge_ids = [int(comic.id) for comic in comics if int(comic.id) != int(canonical.id)]
    chapters, spread, ratio = _chapter_metrics(comics)
    signals = _compute_signals(
        group, min_key_ratio, chapter_spread_max, chapter_ratio_min
    )
    risk = _assess_risk(group, signals, max_group_size, min_key_ratio)
    return {
        "group_id": group.identity_key,
        "reason": group.reason,
        "cross_type": group.cross_type,
        "com_type": int(canonical.com_type),
        "com_types": sorted({int(comic.com_type) for comic in comics}),
        "canonical_id": int(canonical.id),
        "canonical_com_type": int(canonical.com_type),
        "merge_ids": merge_ids,
        "ids": [int(comic.id) for comic in comics],
        "titles": [comic.get_titles() for comic in comics],
        "identity_keys": [str(comic.identity_key or "") for comic in comics],
        "current_chap": chapters,
        "chapter_spread": spread,
        "chapter_ratio": round(ratio, 4),
        "signals": signals,
        "risk": risk,
        "approved": risk == "low",
    }


def _merge_plan_groups(
    session,
    title_like: str,
    min_key_ratio: float,
    merge_cross_type: bool,
) -> List[DuplicateGroup]:
    groups = _load_duplicate_groups(session, title_like)
    title_prefix_groups = _load_title_prefix_duplicate_groups(
        session, title_like, min_key_ratio
    )
    cross_type_groups = (
        _load_cross_type_duplicate_groups(session, title_like, min_key_ratio)
        if merge_cross_type
        else []
    )

    clear_groups = [group for group in groups if not group.ambiguous]
    ambiguous_groups = [group for group in groups if group.ambiguous]
    for group in ambiguous_groups:
        if all(_is_non_novel_comic(comic) for comic in group.comics):
            group.reason = "identity_key_ambiguous_cross_type"

    merge_groups = clear_groups + title_prefix_groups + cross_type_groups
    if merge_cross_type:
        merge_groups += [
            group
            for group in ambiguous_groups
            if all(_is_non_novel_comic(comic) for comic in group.comics)
        ]

    deduped: Dict[tuple[int, ...], DuplicateGroup] = {}
    for group in merge_groups:
        comics = _dedupe_comics_by_id(group.comics)
        if len(comics) < 2:
            continue
        key = tuple(sorted(int(comic.id) for comic in comics))
        existing = deduped.get(key)
        if existing is None or group.reason == "identity_key":
            deduped[key] = DuplicateGroup(
                identity_key=group.identity_key,
                comics=comics,
                reason=group.reason,
            )
    return list(deduped.values())


def _export_plan(
    path: Path,
    entries: List[Dict[str, Any]],
) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "group_count": len(entries),
        "low_risk_count": sum(1 for entry in entries if entry["risk"] == "low"),
        "review_count": sum(1 for entry in entries if entry["risk"] == "review"),
        "cross_type_count": sum(1 for entry in entries if entry["cross_type"]),
        "groups": entries,
    }
    path.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    csv_path = path.with_suffix(".csv")
    with csv_path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=[
                "group_id",
                "reason",
                "cross_type",
                "risk",
                "approved",
                "canonical_id",
                "merge_ids",
                "com_types",
                "chapter_spread",
                "chapter_ratio",
                "signals",
            ],
        )
        writer.writeheader()
        for entry in entries:
            writer.writerow(
                {
                    "group_id": entry["group_id"],
                    "reason": entry["reason"],
                    "cross_type": entry["cross_type"],
                    "risk": entry["risk"],
                    "approved": entry["approved"],
                    "canonical_id": entry["canonical_id"],
                    "merge_ids": ",".join(str(value) for value in entry["merge_ids"]),
                    "com_types": ",".join(str(value) for value in entry["com_types"]),
                    "chapter_spread": entry["chapter_spread"],
                    "chapter_ratio": entry["chapter_ratio"],
                    "signals": json.dumps(entry["signals"]),
                }
            )


def _load_import_plan(path: Path) -> List[DuplicateGroup]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    groups = payload.get("groups", payload)
    if isinstance(groups, dict):
        groups = groups.get("groups", [])
    result: List[DuplicateGroup] = []
    with Session() as session:
        for entry in groups:
            if not entry.get("approved", False):
                continue
            ids = [int(entry["canonical_id"])] + [
                int(value) for value in entry.get("merge_ids", [])
            ]
            comics = (
                session.query(ComicDB)
                .filter(ComicDB.id.in_(ids), ComicDB.deleted == 0)
                .order_by(ComicDB.id)
                .all()
            )
            if len(comics) < 2:
                continue
            result.append(
                DuplicateGroup(
                    identity_key=str(entry.get("group_id", f"import:{ids}")),
                    comics=_dedupe_comics_by_id(comics),
                    reason=str(entry.get("reason", "import_plan")),
                )
            )
    return result


def _active_group_comics(session, group: DuplicateGroup) -> List[ComicDB]:
    comics: List[ComicDB] = []
    for comic in _dedupe_comics_by_id(group.comics):
        row = session.get(ComicDB, int(comic.id))
        if row is None or int(row.deleted) != 0:
            continue
        comics.append(row)
    return sorted(comics, key=lambda row: int(row.id))


def _merge_group(session, group: DuplicateGroup) -> int:
    comics = _active_group_comics(session, group)
    if len(comics) < 2:
        return 0

    canonical = comics[0]
    merged_count = 0

    for duplicate in comics[1:]:
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

        if canonical.com_type == int(Types.Unknown) and duplicate.com_type != int(
            Types.Unknown
        ):
            canonical.com_type = duplicate.com_type
        if canonical.status == int(Statuses.Unknown) and duplicate.status != int(
            Statuses.Unknown
        ):
            canonical.status = duplicate.status

        canonical.author = _prefer_text(canonical.author, duplicate.author)
        canonical.description = _prefer_text(canonical.description, duplicate.description)
        canonical.cover, canonical.cover_visible = _prefer_cover(canonical, duplicate)

        session.delete(duplicate)
        merged_count += 1

    canonical.normalize_titles()
    canonical.refresh_identity_key()
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
        help="Include cross-type identity_key groups when applying without an import plan.",
    )
    parser.add_argument(
        "--merge-cross-type",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="Include cross-type title duplicate groups (default: on).",
    )
    parser.add_argument(
        "--normalize-all-titles",
        action="store_true",
        help=(
            "Normalize the full catalog title list to sentence-case storage and "
            "smart-quote cleanup, even for rows that are already unique."
        ),
    )
    parser.add_argument(
        "--title-like",
        default="",
        help="Restrict duplicate repair to comics whose stored titles contain this text.",
    )
    parser.add_argument(
        "--export-plan",
        metavar="PATH",
        default="",
        help="Write a JSON (+ CSV) merge plan for human review.",
    )
    parser.add_argument(
        "--import-plan",
        metavar="PATH",
        default="",
        help="Apply only groups marked approved in a merge plan JSON file.",
    )
    parser.add_argument(
        "--apply-approved-only",
        action="store_true",
        help="When exporting, only auto-approve low-risk groups (default: low risk pre-approved).",
    )
    parser.add_argument(
        "--min-key-ratio",
        type=float,
        default=DEFAULT_MIN_KEY_RATIO,
        help="Minimum shorter/longer title key ratio for prefix grouping.",
    )
    parser.add_argument(
        "--max-group-size",
        type=int,
        default=DEFAULT_MAX_GROUP_SIZE,
        help="Groups larger than this are export-only review risk.",
    )
    parser.add_argument(
        "--chapter-spread-max",
        type=int,
        default=DEFAULT_CHAPTER_SPREAD_MAX,
        help="Chapter spread threshold for chapters_close signal.",
    )
    parser.add_argument(
        "--chapter-ratio-min",
        type=float,
        default=DEFAULT_CHAPTER_RATIO_MIN,
        help="Min/max chapter ratio threshold when max chapter is at least 20.",
    )
    args = parser.parse_args()

    with Session() as session:
        merge_groups = _merge_plan_groups(
            session,
            args.title_like,
            args.min_key_ratio,
            args.merge_cross_type,
        )
        plan_entries = [
            _group_plan_entry(
                group,
                args.min_key_ratio,
                args.max_group_size,
                args.chapter_spread_max,
                args.chapter_ratio_min,
            )
            for group in merge_groups
        ]

        if args.export_plan:
            export_path = Path(args.export_plan)
            if args.apply_approved_only:
                for entry in plan_entries:
                    entry["approved"] = entry["risk"] == "low"
            _export_plan(export_path, plan_entries)
            print(f"Wrote merge plan: {export_path}")
            print(f"Wrote merge CSV: {export_path.with_suffix('.csv')}")
            print(f"Groups: {len(plan_entries)}")
            print(f"Low risk: {sum(1 for e in plan_entries if e['risk'] == 'low')}")
            print(f"Review: {sum(1 for e in plan_entries if e['risk'] == 'review')}")
            print(f"Cross-type: {sum(1 for e in plan_entries if e['cross_type'])}")
            if not args.apply:
                return 0

        groups = _load_duplicate_groups(session, args.title_like)
        title_prefix_groups = _load_title_prefix_duplicate_groups(
            session, args.title_like, args.min_key_ratio
        )
        cross_type_groups = (
            _load_cross_type_duplicate_groups(session, args.title_like, args.min_key_ratio)
            if args.merge_cross_type
            else []
        )
        clear_groups = [group for group in groups if not group.ambiguous]
        ambiguous_groups = [group for group in groups if group.ambiguous]

        print(f"Duplicate identity groups: {len(groups)}")
        print(f"Title prefix duplicate groups: {len(title_prefix_groups)}")
        print(f"Cross-type duplicate groups: {len(cross_type_groups)}")
        if args.title_like:
            print(f"Title filter: {args.title_like}")
        print(f"Clear merge groups: {len(clear_groups)}")
        print(f"Ambiguous review groups: {len(ambiguous_groups)}")
        print(f"Unified merge plan groups: {len(merge_groups)}")
        title_normalizations = _load_catalog_title_normalizations(session)
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

        if args.import_plan:
            merge_groups = _load_import_plan(Path(args.import_plan))
            print(f"Import plan groups to apply: {len(merge_groups)}")
        else:
            merge_groups = clear_groups
            if args.merge_cross_type:
                merge_groups += cross_type_groups
                merge_groups += [
                    group
                    for group in ambiguous_groups
                    if all(_is_non_novel_comic(comic) for comic in group.comics)
                ]
            if args.merge_ambiguous:
                merge_groups += ambiguous_groups
            if args.title_like:
                merge_groups += title_prefix_groups
            deduped_apply: Dict[tuple[int, ...], DuplicateGroup] = {}
            for group in merge_groups:
                comics = _dedupe_comics_by_id(group.comics)
                if len(comics) < 2:
                    continue
                key = tuple(sorted(int(comic.id) for comic in comics))
                deduped_apply[key] = DuplicateGroup(
                    identity_key=group.identity_key,
                    comics=comics,
                    reason=group.reason,
                )
            merge_groups = list(deduped_apply.values())
            print(f"Direct apply merge groups: {len(merge_groups)}")

        backup_dir = _backup_storage()
        merged_rows = 0
        for group in sorted(
            merge_groups,
            key=lambda item: min(int(comic.id) for comic in item.comics),
        ):
            merged_rows += _merge_group(session, group)
            session.flush()
        normalized_rows = 0
        if args.normalize_all_titles:
            for comic, normalized_titles in title_normalizations:
                comic.set_titles(normalized_titles)
                normalized_rows += 1

        session.commit()
        rebuild_json_backup_from_db()

        remaining_groups = _load_duplicate_groups(session, args.title_like)
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
