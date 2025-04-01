"""
Manhua Plus scraper module.

This module handles scraping comic information from Manhua Plus website.
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
PUBLISHER = Publishers.ManhuaPlus
DEFAULT_COMIC_TYPE = 'manhua'
DEFAULT_STATUS = 'ongoing'

# CSS Selectors
COMIC_CLASS = 'page-item-detail'
COMIC_INFO_CLASS = 'item-summary'
TITLE_CLASS = 'post-title'
CHAPTER_CLASS = 'chapter-item'


def extract_comic_info(comic_div: Tag) -> Optional[ScrapedComic]:
    """
    Extract comic information from a comic detail div.

    Args:
        comic_div: BeautifulSoup Tag containing comic information

    Returns:
        ScrapedComic object if extraction successful, None otherwise
    """
    title = 'Unknown'
    try:
        # Extract cover image
        cover = comic_div.div.a.img['data-src']

        # Extract comic info div
        comic_info = comic_div.select(f"div.{COMIC_INFO_CLASS}")[0]

        # Extract title
        title_div = comic_info.select(f"div.{TITLE_CLASS}")[0]
        title = title_div.h3.a.text.strip()

        # Extract chapter information
        chapter_items = comic_info.find_all(
            'div', attrs={'class': CHAPTER_CLASS})
        if not chapter_items:
            log.warning('No chapters found for comic: %s', title)
            return None

        # Extract latest chapter number
        chapter = chapter_items[0].span.a.text.strip()

        return ScrapedComic(
            chapter=chapter,
            title=title,
            cover_url=cover,
            com_type=DEFAULT_COMIC_TYPE,
            status=DEFAULT_STATUS
        )

    except (ValueError, IndexError, KeyError, AttributeError) as error:
        log.error('Failed to extract comic info for %s: %s', title, error)
        return None


async def scrape_manhuaplus(url: str, session: Session) -> None:
    """
    Scrape comics from Manhua Plus website.

    Args:
        url: URL of the Manhua Plus page to scrape
    """
    soup = await scrape_url(url)

    # Find all comic detail divs
    comic_divs = soup.find_all('div', attrs={'class': COMIC_CLASS})
    if not comic_divs:
        log.error('No comics found on page: %s', url)
        return

    # Process each comic div
    for comic_div in comic_divs:
        comic = extract_comic_info(comic_div)
        if comic:
            await register_comic(comic, PUBLISHER, session)
