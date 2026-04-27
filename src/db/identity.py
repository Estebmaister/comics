from __future__ import annotations

import re
from typing import Iterable, List, Sequence

from helpers.text import normalize_text

NOVEL_TYPE = 4
NOVEL_SUFFIX = " - novel"
NOVEL_IDENTITY_PREFIX = "novel:"
SERIES_IDENTITY_PREFIX = "series:"

_NOVEL_MARKER_RE = re.compile(r"\(\s*novel\s*\)|-\s*novel\s*$", re.IGNORECASE)
_LEADING_ARTICLE_RE = re.compile(r"^(the|a|an)\s+", re.IGNORECASE)
_NON_MATCH_CHAR_RE = re.compile(r"[^a-z0-9]+")


def split_title_values(titles: str | Sequence[str] | None) -> List[str]:
    if titles is None:
        return []
    if isinstance(titles, str):
        return [title for title in titles.split("|")]
    return [str(title) for title in titles]


def strip_novel_marker(title: str) -> str:
    normalized = normalize_text(title)
    if not normalized:
        return ""
    stripped = _NOVEL_MARKER_RE.sub("", normalized)
    return normalize_text(stripped)


def title_has_novel_marker(title: str) -> bool:
    return bool(_NOVEL_MARKER_RE.search(normalize_text(title)))


def is_novel_identity(title: str, com_type: int | None) -> bool:
    return int(com_type or 0) == NOVEL_TYPE or title_has_novel_marker(title)


def normalize_title_for_storage(
    title: str,
    com_type: int | None = 0,
    *,
    primary: bool = False,
) -> str:
    cleaned = strip_novel_marker(title)
    if not cleaned:
        return ""
    if primary and is_novel_identity(title, com_type):
        cleaned = f"{cleaned}{NOVEL_SUFFIX}"
    return cleaned.capitalize()


def normalize_title_variants(
    titles: str | Sequence[str] | None,
    com_type: int | None = 0,
) -> List[str]:
    normalized_titles: List[str] = []
    for index, raw_title in enumerate(split_title_values(titles)):
        normalized = normalize_title_for_storage(
            raw_title, com_type, primary=index == 0
        )
        if normalized and normalized not in normalized_titles:
            normalized_titles.append(normalized)
    return normalized_titles


def primary_title_from_titles(titles: str | Sequence[str] | None) -> str:
    for title in split_title_values(titles):
        normalized = normalize_text(title)
        if normalized:
            return normalized
    return ""


def normalize_primary_title(title: str) -> str:
    return strip_novel_marker(title).lower()


def title_match_key(title: str) -> str:
    normalized = normalize_primary_title(title)
    normalized = _LEADING_ARTICLE_RE.sub("", normalized)
    return _NON_MATCH_CHAR_RE.sub("", normalized)


def titles_are_prefix_match(incoming_title: str, stored_title: str) -> bool:
    incoming_key = title_match_key(incoming_title)
    stored_key = title_match_key(stored_title)
    return bool(incoming_key and stored_key and stored_key.startswith(incoming_key))


def build_identity_key(primary_title: str, com_type: int | None = 0) -> str:
    normalized_title = normalize_primary_title(primary_title)
    if not normalized_title:
        return ""
    identity_prefix = (
        NOVEL_IDENTITY_PREFIX
        if is_novel_identity(primary_title, com_type)
        else SERIES_IDENTITY_PREFIX
    )
    return f"{identity_prefix}{normalized_title}"


def build_identity_key_from_titles(
    titles: str | Sequence[str] | None,
    com_type: int | None = 0,
) -> str:
    return build_identity_key(primary_title_from_titles(titles), com_type)


def merge_unique_values(current_values: Iterable[int], incoming_values: Iterable[int]) -> List[int]:
    merged: List[int] = []
    for value in list(current_values) + list(incoming_values):
        int_value = int(value)
        if int_value not in merged:
            merged.append(int_value)
    return merged
