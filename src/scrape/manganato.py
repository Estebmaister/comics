"""
Manganato scraper module.

This module handles scraping comic information from Manganato website.
It extracts chapter numbers, titles, cover images, authors and other metadata.
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
PUBLISHER = Publishers.Manganato
DEFAULT_COMIC_TYPE = 'manhwa'
DEFAULT_STATUS = 'ongoing'
COMIC_ITEM_CLASS = 'itemupdate first'


def extract_comic_info(comic_div: Tag) -> Optional[ScrapedComic]:
    """
    Extract comic information from a comic item div.

    Args:
        comic_div: BeautifulSoup Tag containing comic information

    Returns:
        ScrapedComic object if extraction successful, None otherwise
    """
    title = 'Unknown'
    try:
        # Extract cover image
        cover = comic_div.a.img['src']

        # Extract comic internal div with title and metadata
        comic_int = comic_div.ul

        # Extract title
        title = comic_int.li.h3.a.text.strip()

        # Extract author (optional)
        try:
            author = comic_int.li.span.text.strip()
        except AttributeError:
            author = ''

        # Extract latest chapter number
        try:
            chap = comic_int.select('li')[1].span.a.text
        except AttributeError:
            log.info('No chapters found for comic: %s', title)
            return None

        return ScrapedComic(
            chapter=chap,
            title=title,
            cover_url=cover,
            com_type=DEFAULT_COMIC_TYPE,
            status=DEFAULT_STATUS,
            author=author
        )

    except (ValueError, IndexError, KeyError, AttributeError) as error:
        log.error('Failed to extract comic info for %s: %a', title, error)
        return None


async def scrape_manganato(url: str, session: Session) -> None:
    """
    Scrape comics from Manganato website.

    Args:
        url: URL of the Manganato page to scrape
    """
    soup = await scrape_url(url)

    # Find all comic item divs
    comic_divs = soup.find_all('div', attrs={'class': COMIC_ITEM_CLASS})
    if not comic_divs:
        log.error('No comics found on page: %s', url)
        return

    # Process each comic div
    for comic_div in comic_divs:
        comic = extract_comic_info(comic_div)
        if comic:
            await register_comic(comic, PUBLISHER, session)
