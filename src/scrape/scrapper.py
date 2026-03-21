"""
Comic scraper module for fetching and processing comic information from various publishers.

This module handles web scraping of comic information, including chapters, titles,
and metadata, and manages their storage in both database and JSON formats.
"""

from __future__ import annotations

import asyncio
import os
import re
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Dict, List, Optional

import cloudscraper
from bs4 import BeautifulSoup
from sqlalchemy.orm import Session

from db import ComicDB, Publishers, Statuses, Types, load_comics
from db.identity import merge_unique_values
from db.repo import comics_by_identity_key, create_comic, rebuild_json_backup_from_db
from helpers.alert import add_alert
from helpers.logger import logger
from helpers.text import normalize_text
from scrape.url_switch import url_switch

# Configure logging
log = logger(__name__)

# Constants
CHAPS_FILE = os.path.join(os.path.dirname(__file__), "../db/chaps.html")
REQUEST_TIMEOUT = 10  # seconds
MINIMUM_COVER_URL_LENGTH = 10

# Initialize scraper with modern browser configuration
scraper = cloudscraper.create_scraper(browser='chrome')

# Publisher-specific configurations
COVER_UPDATE_PUBLISHERS = {
    Publishers.Asura,
    Publishers.FlameScans,
    Publishers.Manganato,
    Publishers.RealmScans,
    Publishers.DemonicScans,
    Publishers.IsekaiScan
}

RESTRICTED_COVER_PUBLISHERS = {
    Publishers.ManhuaPlus,
    Publishers.ReaperScans,
    Publishers.Manganato
}


@dataclass
class ScrapedComic:
    """Data class to hold normalized comic information."""
    chapter: str
    title: str
    cover_url: str
    com_type: str
    status: str
    author: str = ''


@dataclass
class DiscoveryRunState:
    seen_publisher_keys: set[tuple[int, str]] = field(default_factory=set)
    lock: asyncio.Lock = field(default_factory=asyncio.Lock)


class _AsyncNoopLock:
    async def __aenter__(self) -> None:
        return None

    async def __aexit__(self, exc_type, exc, tb) -> bool:
        return False


async def scrape_url(url: str, debug_pattern: str = ' ') -> BeautifulSoup:
    """
    Scrape content from a URL with error handling and optional debug output.

    Args:
        url: Target URL to scrape
        debug_pattern: Pattern to trigger debug output saving

    Returns:
        BeautifulSoup object containing parsed HTML
        Empty BeautifulSoup object if scraping fails
    """
    try:
        with scraper.get(url, timeout=REQUEST_TIMEOUT) as response:
            if response.status_code != 200:
                log.warning('HTTP %s: Failed to fetch %s',
                            response.status_code, url)
            soup = BeautifulSoup(response.text, 'html.parser')
            if debug_pattern in url:
                await _save_debug_output(soup)
            return soup

    except Exception as err:
        log.warning(
            'Error %s while fetching %s: %s',
            type(err).__name__,
            url,
            str(err)
        )
        return BeautifulSoup('', 'html.parser')


def _save_debug_output(soup: BeautifulSoup) -> None:
    """Save scraped content to debug file."""
    with open(CHAPS_FILE, 'w+') as file:
        file.write(soup.prettify())


async def register_comic(
    scraped_comic: ScrapedComic,
    publisher: Publishers,
    session: Session,
    run_state: DiscoveryRunState | None = None,
) -> None:
    """
    Register or update a comic in the database and JSON storage.

    Args:
        scraped_comic: Scraped comic information
        publisher: Publisher information
    """
    # Clean and normalize input parameters
    normalized_comic = _normalize_comic_data(scraped_comic, publisher)
    if not normalized_comic:
        return

    lock = run_state.lock if run_state is not None else _AsyncNoopLock()
    publisher_key = (int(publisher), normalized_comic.identity_key)

    async with lock:
        if run_state is not None and publisher_key in run_state.seen_publisher_keys:
            log.debug(
                'Skipping duplicate discovery within run for %s (%s)',
                normalized_comic.titles,
                publisher.name,
            )
            return
        if run_state is not None:
            run_state.seen_publisher_keys.add(publisher_key)

        try:
            with session.begin_nested():
                db_comics = comics_by_identity_key(
                    normalized_comic.identity_key, session
                )
                if not db_comics:
                    create_comic(normalized_comic, session)
                    return

                if len(db_comics) > 1:
                    log.warning(
                        'Found %d duplicates for identity %s, using canonical comic ID %s',
                        len(db_comics),
                        normalized_comic.identity_key,
                        db_comics[0].id,
                    )

                await _update_existing_comic(
                    db_comics[0], normalized_comic, publisher
                )
                session.flush()
        except Exception as error:
            if run_state is not None:
                run_state.seen_publisher_keys.discard(publisher_key)
            rebuild_json_backup_from_db(session, persist_file=False)
            log.error(
                'Failed to register comic %s: %s',
                normalized_comic.titles,
                error,
            )


async def _update_existing_comic(
    db_comic: ComicDB,
    comic: ComicDB,
    publisher: Publishers
) -> None:
    """Update an existing comic entry with new information."""
    json_comic = _ensure_json_record(db_comic)[0]

    # Update publisher if new
    if publisher not in db_comic.get_published_in():
        merged_publishers = merge_unique_values(
            [int(pub) for pub in db_comic.get_published_in()],
            [int(publisher)],
        )
        db_comic.set_published_in(merged_publishers)
        json_comic['published_in'] = db_comic.get_published_in()
        log.info('%s: Added new publisher: %s',
                 db_comic.titles, publisher.name)

    # Update chapter if newer
    if comic.current_chap > db_comic.current_chap:
        _update_chapter_info(db_comic, json_comic, comic.current_chap)

    # Update metadata
    _update_metadata(
        db_comic, json_comic,
        author=comic.author, com_type=comic.com_type, status=comic.status
    )

    # Update cover if needed
    await _update_cover_if_needed(db_comic, json_comic, comic.cover, publisher)


def _ensure_json_record(db_comic: ComicDB) -> List[Dict]:
    """Ensure a JSON record exists for the database entry."""
    json_comics = [
        comic for comic in load_comics if db_comic.id == comic["id"]]
    if not json_comics:
        log.debug(
            '%s was not found in JSON backup (ID: %s)',
            db_comic.titles, db_comic.id
        )
        json_comics = [db_comic.toJSON()]
        load_comics.append(json_comics[0])
    return json_comics


def _update_chapter_info(
    db_comic: ComicDB,
    json_comic: Dict,
    new_chap: int
) -> None:
    """Update chapter information and trigger alerts if needed."""
    # Update current chapter
    db_comic.current_chap = new_chap
    json_comic['current_chap'] = new_chap
    # Update last update
    timestamp = int(time.time())
    db_comic.last_update = timestamp
    json_comic['last_update'] = datetime.fromtimestamp(
        timestamp, tz=timezone.utc).isoformat()
    # Trigger alerts only for tracked comics
    if not db_comic.track:
        return
    # Only trigger alerts for new chapters within 4 chapters of last read
    if db_comic.viewed_chap > new_chap - 4:
        add_alert(db_comic.get_titles()[0], str(
            new_chap), db_comic.get_published_in())


def _update_metadata(
    db_comic: ComicDB,
    json_comic: Dict,
    author: str = '',
    com_type: Types = Types.Unknown,
    status: Statuses = Statuses.Unknown
) -> None:
    """Update comic metadata if new information is available."""
    if author and not db_comic.author:
        db_comic.author = author
        json_comic['author'] = author

    if com_type != Types.Unknown and db_comic.com_type == Types.Unknown:
        db_comic.com_type = com_type
        json_comic['com_type'] = com_type

    if status != Statuses.Unknown:
        db_comic.status = status
        json_comic['status'] = status


async def _update_cover_if_needed(
    db_comic: ComicDB,
    json_comic: Dict,
    cover: str,
    publisher: Publishers
) -> None:
    """
    Update comic cover URL based on publisher rules.

    Some publishers have specific rules about when covers should be updated,
    either due to load restrictions or reliability of cover URLs.
    """
    if not cover or db_comic.cover == cover:
        return

    # Update if no existing cover
    if not db_comic.cover:
        db_comic.cover = cover
        json_comic['cover'] = cover
        return

    # Update if current publisher is not restricted but comic is available
    # on restricted publishers
    if (publisher not in RESTRICTED_COVER_PUBLISHERS and
        any(pub in db_comic.get_published_in()
            for pub in RESTRICTED_COVER_PUBLISHERS)):
        db_comic.cover = cover
        json_comic['cover'] = cover
        return

    # Update for specific publishers that need regular cover updates
    if publisher in COVER_UPDATE_PUBLISHERS and publisher not in RESTRICTED_COVER_PUBLISHERS:
        db_comic.cover = cover
        json_comic['cover'] = cover

    # Update for restricted publishers when there is no other publisher
    if publisher in RESTRICTED_COVER_PUBLISHERS and not any(
        pub not in RESTRICTED_COVER_PUBLISHERS for pub in db_comic.get_published_in()
    ):
        db_comic.cover = cover
        json_comic['cover'] = cover


def _normalize_comic_data(scraped_comic: ScrapedComic, publisher: Publishers) -> Optional[ComicDB]:
    """
    Clean and normalize comic data for consistency.

    Args:
        scraped_comic: ScrapedComic object containing comic data
        publisher: Publisher enum for URL processing

    Returns:
        ComicDB object containing normalized comic information
    """
    # Extract chapter number
    try:
        chapter_num = int(re.findall(r'\d+', scraped_comic.chapter)[0])
    except (ValueError, IndexError) as error:
        log.error('Failed to parse chapter number "%s" for "%s": %s',
                  scraped_comic.chapter, scraped_comic.title, error)
        return None

    # Clean title
    clean_title = normalize_text(scraped_comic.title)

    # Normalize cover URL
    cover_url = _parse_cover_url(scraped_comic.cover_url, publisher)

    # Parse type
    type_parsed = _parse_type(scraped_comic.com_type)

    # Parse status
    status_parsed = _parse_status(scraped_comic.status)

    return ComicDB(
        id=None,
        titles=clean_title,
        current_chap=chapter_num,
        cover=cover_url,
        com_type=type_parsed,
        status=status_parsed,
        published_in=publisher,
        author=scraped_comic.author
    )


def _parse_cover_url(cover: str, publisher: Publishers) -> str:
    """Normalize cover URL based on its format and publisher."""
    if 'http' in cover:
        return cover[cover.find('http'):]
    elif cover.startswith('/'):
        base_url = url_switch.get(publisher.name, [''])[0]
        return base_url + cover

    if len(cover) < MINIMUM_COVER_URL_LENGTH:
        log.error('Invalid cover URL (%s) for publisher %s',
                  cover, publisher.name)
        return ''

    return cover


def _parse_status(status_text: str) -> Statuses:
    """Convert status text to Status enum."""
    status_text = status_text.strip().lower()

    STATUS_MAP = {
        'completed': Statuses.Completed,
        'ongoing': Statuses.OnAir,
        'hiatus': Statuses.Break,
        'season end': Statuses.Break,
        'dropped': Statuses.Dropped
    }

    return STATUS_MAP.get(status_text, Statuses.Unknown)


def _parse_type(type_text: str) -> Types:
    """Convert type text to Type enum."""
    type_text = type_text.strip().replace('NEW ', '').lower()

    TYPE_MAP = {
        'manga': Types.Manga,
        'manhua': Types.Manhua,
        'manhwa': Types.Manhwa,
        'webtoon': Types.Manhwa,
        'novel': Types.Novel
    }

    return TYPE_MAP.get(type_text, Types.Unknown)
