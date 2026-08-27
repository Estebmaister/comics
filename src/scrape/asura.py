"""
Asura scans scraper module.

This module handles scraping comic information from Asura Scans website.

The site is built with Astro and ships its comic data as serialized JSON inside
``astro-island`` custom elements (in their ``props`` attribute). The payload uses
Astro's island encoding where every value is wrapped as ``[type, data]``:

    0 -> raw value (objects are decoded recursively)
    1 -> array of encoded values
    3 -> Date string
    others (RegExp, Map, Set, BigInt, URL, typed arrays, Infinity)

We decode that payload and pull comics out of the islands that carry catalog
data (latest updates, hero carousel, trending, popular sidebar).
"""

import json
import re
from typing import Any, Dict, List, Optional

from sqlalchemy.orm import Session

from db import Publishers
from helpers.logger import logger
from scrape.scrapper import (DiscoveryRunState, ScrapedComic, register_comic,
                             scrape_url)

# Configure logging
log = logger(__name__)

# Publisher-specific constants
PUBLISHER = Publishers.Asura
DEFAULT_COMIC_TYPE = 'manhwa'
DEFAULT_STATUS = 'ongoing'

# Islands that carry catalog data and the key holding the comic collection.
# ``chapters`` islands describe latest chapter releases; ``items`` islands
# describe full series entries.
ISLAND_COLLECTION_KEYS = ('chapters', 'items', 'initialSeries')


def _astro_decode(value: Any) -> Any:
    """Decode a single Astro-encoded ``[type, data]`` value."""
    if not isinstance(value, list) or len(value) != 2:
        return value

    type_id, data = value
    if type_id == 0:
        return _astro_decode_object(data)
    if type_id == 1:
        return [_astro_decode(item) for item in data]
    if type_id == 4:  # Map -> dict
        return {_astro_decode(k): _astro_decode(v) for k, v in data}
    if type_id == 5:  # Set -> list
        return [_astro_decode(item) for item in data]
    # 2 RegExp, 3 Date, 6 BigInt, 7 URL, 8-11 typed/Infinity: keep raw payload
    return data


def _astro_decode_object(data: Any) -> Any:
    """Decode an Astro object payload, recursing into each property value."""
    if isinstance(data, dict):
        return {key: _astro_decode(val) for key, val in data.items()}
    return data


def _parse_island_props(raw_props: str) -> Dict[str, Any]:
    """Parse and decode the ``props`` attribute of an astro-island element."""
    try:
        parsed = json.loads(raw_props)
    except (json.JSONDecodeError, TypeError):
        return {}
    decoded = _astro_decode_object(parsed)
    return decoded if isinstance(decoded, dict) else {}


def _comic_from_entry(entry: Dict[str, Any]) -> Optional[ScrapedComic]:
    """Build a ScrapedComic from a decoded island entry."""
    if not isinstance(entry, dict):
        return None

    title = entry.get('comic_name') or entry.get('title')
    cover = entry.get('comic_cover') or entry.get('cover_url') or entry.get('cover') or ''
    com_type = entry.get('type') or DEFAULT_COMIC_TYPE
    status = entry.get('status') or DEFAULT_STATUS

    # ``chapters`` islands expose the chapter number directly; series islands
    # only expose the latest chapter count.
    chapter = entry.get('number')
    if chapter is None:
        chapter = entry.get('chapter_count')
    if chapter is None:
        name = entry.get('name')
        if name is not None and re.search(r'\d', str(name)):
            chapter = name

    if not title or chapter is None:
        return None

    return ScrapedComic(
        chapter=str(chapter),
        title=str(title),
        cover_url=str(cover),
        com_type=str(com_type),
        status=str(status),
    )


def extract_comics_from_props(props: Dict[str, Any]) -> List[ScrapedComic]:
    """Extract every comic found in a decoded island props dictionary."""
    comics: List[ScrapedComic] = []
    for key in ISLAND_COLLECTION_KEYS:
        collection = props.get(key)
        if not isinstance(collection, list):
            continue
        for entry in collection:
            comic = _comic_from_entry(entry)
            if comic:
                comics.append(comic)
    return comics


async def scrape_asura(
    url: str,
    session: Session,
    run_state: DiscoveryRunState | None = None,
) -> None:
    """
    Scrape comics from Asura Scans website.

    Args:
        url: URL of the Asura Scans page to scrape
        session: Active database session
        run_state: Optional shared discovery state for de-duplication
    """
    soup = await scrape_url(url)

    islands = soup.find_all('astro-island', attrs={'props': True})
    if not islands:
        log.error('No comics found on page: %s', url)
        return

    # Collect comics across every catalog-bearing island, de-duplicating by
    # title to avoid redundant work for entries that appear in several sections.
    seen_titles: set[str] = set()
    comics: List[ScrapedComic] = []
    for island in islands:
        props = _parse_island_props(island.get('props', ''))
        if not props:
            continue
        for comic in extract_comics_from_props(props):
            key = comic.title.strip().lower()
            if key in seen_titles:
                continue
            seen_titles.add(key)
            comics.append(comic)

    if not comics:
        log.error('No comics found on page: %s', url)
        return

    for comic in comics:
        await register_comic(comic, PUBLISHER, session, run_state)
