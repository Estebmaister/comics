"""
Demonic scans scraper module.

This module handles scraping comic information from Demonic Scans website.
It extracts chapter numbers, titles, cover images and other metadata.
"""

from typing import Optional

from bs4 import Tag

from db import Publishers
from helpers.logger import logger
from scrape.scrapper import ScrapedComic, register_comic, scrape_url

# Configure logging
log = logger(__name__)

# Publisher-specific constants
PUBLISHER = Publishers.DemonicScans
DEFAULT_COMIC_TYPE = 'manhwa'
DEFAULT_STATUS = 'ongoing'
COMIC_GRID_CLASS = 'updates-element border-box'


def extract_comic_info(comic_div: Tag) -> Optional[ScrapedComic]:
    """
    Extract comic information from a comic grid div.

    Args:
        comic_div: BeautifulSoup Tag containing comic information

    Returns:
        ScrapedComic object if extraction successful, None otherwise
    """
    title = 'Unknown'
    try:
        # Extract cover image
        cover = comic_div.div.div.a.img['src']

        # Extract comic internal div with title and chapters
        comic_int = comic_div.div.select('div')[1]

        # Extract and clean title
        title = comic_int.h2.a.text.strip()

        # Extract chapter spans
        chap_int = comic_int.find_all('a')
        if not chap_int:
            log.debug('Skipping recommended comic: %s', title)
            return None

        # Extract chapter number
        chap = chap_int[1].text

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


async def scrape_demonic(url: str) -> None:
    """
    Scrape comics from DemonicScans website.

    Args:
        url: URL of the DemonicScans page to scrape
    """
    soup = await scrape_url(url)

    # Find all comic grid divs
    comic_divs = soup.find_all('div', attrs={'class': COMIC_GRID_CLASS})
    if not comic_divs:
        log.error('No comics found on page: %s', url)
        return

    # Process each comic div
    for comic_div in comic_divs:
        comic = extract_comic_info(comic_div)
        if comic:
            await register_comic(comic, PUBLISHER)
