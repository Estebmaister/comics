"""
Realm Scans scraper module.

This module handles scraping comic information from Realm Scans website.
It extracts chapter numbers, titles, cover images, status and other metadata.
"""

from typing import Optional

from bs4 import Tag

from db import Publishers
from helpers.logger import logger
from scrape.scrapper import ScrapedComic, register_comic, scrape_url

# Configure logging
log = logger(__name__)

# Publisher-specific constants
PUBLISHER = Publishers.RealmScans
DEFAULT_COMIC_TYPE = 'manhwa'
DEFAULT_STATUS = 'ongoing'

# CSS Selectors
COMIC_CLASS = 'uta'
COMIC_INFO_CLASS = 'luf'


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
        # Extract cover image and status
        comic_link = comic_div.div.a
        cover = comic_link.img['src']
        status = comic_link.div.span.text.strip()

        # Extract comic info div
        comic_info = comic_div.select(f'div.{COMIC_INFO_CLASS}')[0]

        # Extract title
        title = comic_info.a.h4.text.strip()

        # Extract chapter information
        chapter_list = comic_info.select('ul')
        if not chapter_list:
            log.debug('Skipping recommended comic: %s', title)
            return None

        # Extract comic type and chapter
        comic_type = chapter_list[0].get('class', [DEFAULT_COMIC_TYPE])[0]
        chapter = chapter_list[0].li.a.text.strip()

        return ScrapedComic(
            chapter=chapter,
            title=title,
            cover_url=cover,
            com_type=comic_type,
            status=status
        )

    except (ValueError, IndexError, KeyError, AttributeError) as error:
        log.error('Failed to extract comic info for %s: %s', title, error)
        return None


async def scrape_realm(url: str) -> None:
    """
    Scrape comics from Realm Scans website.

    Args:
        url: URL of the Realm Scans page to scrape
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
            await register_comic(comic, PUBLISHER)
