"""
Flame Scans scraper module.

This module handles scraping comic information from Flame Scans website.
It extracts chapter numbers, titles, cover images and other metadata.
"""

from typing import Optional

from bs4 import Tag
from sqlalchemy.orm import Session

from db import Publishers
from helpers.logger import logger
from scrape.scrapper import ScrapedComic, register_comic, scrape_url

# Configure logging
log = logger(__name__)

# Publisher-specific constants
PUBLISHER = Publishers.FlameScans
DEFAULT_COMIC_TYPE = 'manhwa'
DEFAULT_STATUS = 'ongoing'

# CSS Selectors
COMIC_CLASS = 'bsx'
COMIC_INFO_CLASS = 'bigor'
TITLE_CLASS = 'tt'
CHAPTER_LIST_CLASS = 'chapter-list'


def extract_comic_info(comic_div: Tag) -> Optional[ScrapedComic]:
    """
    Extract comic information from a comic box div.

    Args:
        comic_div: BeautifulSoup Tag containing comic information

    Returns:
        ScrapedComic object if extraction successful, None otherwise
    """
    title = 'Unknown'
    try:
        # Extract cover image
        cover = comic_div.a.div.img['src']

        # Extract comic info div
        comic_info = comic_div.select(f'div.{COMIC_INFO_CLASS}')[0]

        # Extract title
        title = comic_info.select(f'div.{TITLE_CLASS}')[0].text.strip()

        # Extract chapter information
        chapter_elements = comic_info.select(f'div.{CHAPTER_LIST_CLASS}')
        if not chapter_elements:
            log.debug('Skipping recommended comic: %s', title)
            return None

        # Extract chapter number
        chap = chapter_elements[0].a.div.div.text.strip()

        return ScrapedComic(
            chapter=chap,
            title=title,
            cover_url=cover,
            com_type=DEFAULT_COMIC_TYPE,
            status=DEFAULT_STATUS
        )

    except (ValueError, IndexError, KeyError, AttributeError) as error:
        log.error('Failed to extract comic info for %s: %s', title, error)
        return None


async def scrape_flame(url: str, session: Session) -> None:
    """
    Scrape comics from Flame Scans website.

    Args:
        url: URL of the Flame Scans page to scrape
    """
    soup = await scrape_url(url)

    # Find all comic box divs
    comic_divs = soup.find_all(class_=COMIC_CLASS)
    if not comic_divs:
        log.error('No comics found on page: %s', url)
        return

    # Process each comic div
    for comic_div in comic_divs:
        comic = extract_comic_info(comic_div)
        if comic:
            await register_comic(comic, PUBLISHER, session)
