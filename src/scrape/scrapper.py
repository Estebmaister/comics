"""
Comic scraper module for fetching and processing comic information from various publishers.

This module handles web scraping of comic information, including chapters, titles,
and metadata, and manages their storage in both database and JSON formats.
"""

from __future__ import annotations

import json
import os
import re
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Dict, List, Optional

import cloudscraper
from bs4 import BeautifulSoup
from sqlalchemy.orm import Session

from db import (ComicDB, Publishers, Statuses, Types, load_comics,
                save_comics_file)
from db.helpers import manage_multi_finds
from db.repo import comics_like_title
from helpers.alert import add_alert
from helpers.logger import logger
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


async def register_comic(scraped_comic: ScrapedComic, publisher: Publishers, session: Session) -> None:
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

    # Fetch existing records
    db_comics = comics_like_title(str(normalized_comic.titles), session)
    json_comics = [
        comic for comic in load_comics if str(normalized_comic.titles) in comic["titles"]
    ]

    # Handle multiple matches
    db_comics, final_title = manage_multi_finds(
        db_comics, int(normalized_comic.com_type), str(normalized_comic.titles))
    normalized_comic.set_titles(final_title)

    if not db_comics:
        await _create_new_comic_entry(session, normalized_comic)
        return

    if len(db_comics) != 1:
        log.error(
            'Found %d matches for title: %s - cannot process',
            len(db_comics), normalized_comic.titles
        )
        for db_comic in db_comics:
            log.error('ID: %s, Titles: %s', db_comic.id, db_comic.titles)
        return

    # Update existing comic
    await _update_existing_comic(db_comics[0], json_comics, normalized_comic, publisher)

    # Save changes in memory
    try:
        session.flush()
    except Exception as e:
        session.rollback()
        log.error('Failed to flush session: %s, rolling back on comic %s', str(
            e), normalized_comic.titles)
        return
    save_comics_file(load_comics)


async def _create_new_comic_entry(session, comic: ComicDB) -> None:
    """Create a new comic entry in both database and JSON storage."""
    session.add(comic)
    session.commit()

    json_entry = comic.toJSON()
    log.info('Created new entry: %s', json.dumps(json_entry))
    load_comics.append(json_entry)
    save_comics_file(load_comics)


async def _update_existing_comic(
    db_comic: ComicDB,
    json_comics: List[Dict],
    comic: ComicDB,
    publisher: Publishers
) -> None:
    """Update an existing comic entry with new information."""
    # Ensure JSON record exists
    if not json_comics:
        json_comics = _ensure_json_record(db_comic)

    json_comic = json_comics[0]

    # Update publisher if new
    if publisher not in db_comic.get_published_in():
        db_comic.published_in += f'|{publisher}'
        json_comic['published_in'].append(publisher)
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
    clean_title = scraped_comic.title.strip().capitalize()
    clean_title = clean_title.replace('(novel)', ' - novel')

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
