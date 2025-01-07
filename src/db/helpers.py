# used in db/repopulate_db.py
from typing import List, Optional, Tuple

from . import ComicDB, Types


def manage_multi_finds(db_comics: List[ComicDB], com_type: int, title: str) -> Tuple[List[ComicDB], str]:
    """Handle multiple comic matches and return appropriate result.

    Args:
        db_comics: List of found comics from database
        com_type: Type of comic being searched
        title: Title being searched

    Returns:
        Tuple containing:
        - List of matched comics (empty for no match, single item for match)
        - Final title (may be modified for novels)
    """
    if not db_comics:
        return [], title

    # Handle single comic case
    if len(db_comics) == 1:
        return _handle_single_comic(db_comics[0], title, com_type)

    # Handle exact title match for multiple comics
    for comic in db_comics:
        if title in comic.get_titles():
            return [comic], comic.titles

    # For exactly two comics, try type and title matching
    if len(db_comics) == 2:
        # Try matching by type
        if matched_comic := _handle_type_match(db_comics, com_type):
            return [matched_comic], matched_comic.titles

        # Try matching by exact title
        if matched_comic := _handle_title_match(db_comics, title):
            return [matched_comic], matched_comic.titles

    # No match found
    return [], title


def _handle_single_comic(db_comic: ComicDB, title: str, com_type: int) -> Tuple[List[ComicDB], str]:
    """Handle case when only one comic is found."""
    if title in db_comic.get_titles():
        return [db_comic], db_comic.titles

    title = _handle_novel_type(db_comic, title, com_type)
    if _find_matching_title(db_comic, title):
        return [db_comic], db_comic.titles

    return [], title


def _handle_novel_type(db_comic: ComicDB, title: str, com_type: int) -> str:
    """Handle special case for novels with same title as comic."""
    if com_type == Types.Novel.value and getattr(db_comic, "com_type") != com_type:
        return title + " - novel"
    return title


def _find_matching_title(db_comic: ComicDB, title: str) -> bool:
    """Check if title matches any in comic's titles."""
    for title_db in db_comic.get_titles():
        if title in title_db and "- novel" not in title_db:
            return True
    return False


def _handle_type_match(comics: List[ComicDB], com_type: int) -> Optional[ComicDB]:
    """Find comic with matching type from two comics."""
    for comic in comics:
        if getattr(comic, "com_type") == com_type:
            return comic
    return None


def _handle_title_match(comics: List[ComicDB], title: str) -> Optional[ComicDB]:
    """Find comic with exact title match from two comics."""
    for comic in comics:
        if title in comic.get_titles():
            return comic
    return None
