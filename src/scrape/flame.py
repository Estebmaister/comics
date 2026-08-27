"""
Flame Scans scraper module.

The site uses Mantine SeriesCard components instead of legacy div.bsx cards.
"""

import re
from typing import Optional

from bs4 import Tag
from sqlalchemy.orm import Session

from db import Publishers
from helpers.logger import logger
from scrape.scrapper import DiscoveryRunState, ScrapedComic, register_comic, scrape_url

log = logger(__name__)

PUBLISHER = Publishers.FlameScans
DEFAULT_COMIC_TYPE = 'manhwa'
DEFAULT_STATUS = 'ongoing'

CHAPTER_RE = re.compile(r'Chapter\s+(\d+)', re.IGNORECASE)
COVER_RE = re.compile(r'uploads%2Fimages%2Fseries%2F\d+%2F[^&"]+')


def _parse_flame_cover(card_html: str) -> str:
    match = COVER_RE.search(card_html)
    if not match:
        return ''
    path = match.group(0).replace('%2F', '/').replace('%3F', '?')
    return f'https://flamecomics.xyz/{path}'


def extract_comic_info(comic_div: Tag) -> Optional[ScrapedComic]:
    """Extract comic information from a SeriesCard chapter container."""
    title = 'Unknown'
    try:
        link = comic_div.find('a', class_=lambda c: c and 'chapterImageLink' in c)
        if not link:
            return None

        title = (link.get('title') or '').strip()
        if not title:
            img = link.find('img')
            title = (img.get('alt') or '').strip() if img else ''
        if not title:
            return None

        card_html = str(comic_div)
        chapter_match = CHAPTER_RE.search(card_html)
        if not chapter_match:
            log.debug('Skipping Flame comic without chapter: %s', title)
            return None

        status = DEFAULT_STATUS
        card_text = comic_div.get_text(' ', strip=True).lower()
        if 'completed' in card_text:
            status = 'completed'

        return ScrapedComic(
            chapter=chapter_match.group(1),
            title=title,
            cover_url=_parse_flame_cover(card_html),
            com_type=DEFAULT_COMIC_TYPE,
            status=status,
        )

    except (ValueError, IndexError, KeyError, AttributeError) as error:
        log.error('Failed to extract comic info for %s: %s', title, error)
        return None


async def scrape_flame(
    url: str,
    session: Session,
    run_state: DiscoveryRunState | None = None,
) -> None:
    """Scrape comics from Flame Scans website."""
    soup = await scrape_url(url)

    comic_divs = soup.find_all(class_=lambda c: c and 'chapterCardContainer' in c)
    if not comic_divs:
        log.error('No comics found on page: %s', url)
        return

    for comic_div in comic_divs:
        comic = extract_comic_info(comic_div)
        if comic:
            await register_comic(comic, PUBLISHER, session, run_state)
