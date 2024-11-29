"""
Comic scraping package initialization.

This module provides functions to scrape comics from various publisher websites.
It coordinates the scraping process across different publishers and manages the scraping workflow.
"""

import asyncio
from typing import Callable, Dict, List, Optional

from bs4 import BeautifulSoup, Tag

from db import Publishers, session
from helpers.logger import logger
from scrape.asura import scrape_asura
from scrape.flame import scrape_flame
from scrape.manganato import scrape_manganato
from scrape.manhuaplus import scrape_manhuaplus
from scrape.realm import scrape_realm
from scrape.scrapper import ScrapedComic, register_comic, scrape_url
from scrape.url_switch import publisher_url_pairs

# Configure logging
log = logger(__name__)


async def func_pending(url: str) -> None:
    """Placeholder for pending scraper implementation."""
    pass


async def site_closed(url: str) -> None:
    """Handler for closed/inactive sites."""
    pass


# Map publisher names to their scraping functions
SCRAPE_FUNCTIONS: Dict[str, Callable] = {
    # Publishers.NightScans.name: 		scrape_nightscans,  # TODO
    # Publishers.ReaperScans.name: 		scrape_reaper,  # TODO
    Publishers.ManhuaPlus.name: 		scrape_manhuaplus,
    Publishers.Asura.name: 				scrape_asura,
    Publishers.FlameScans.name: 		scrape_flame,
    Publishers.RealmScans.name: 		scrape_realm,
    Publishers.Manganato.name: 			scrape_manganato,
    Publishers.LeviatanScans.name: 	    site_closed,
    Publishers.LuminousScans.name: 	    func_pending,
    Publishers.IsekaiScan.name: 		func_pending,
    Publishers.VoidScans.name: 			func_pending,
    Publishers.ResetScans.name: 		func_pending,
    Publishers.DrakeScans.name: 		func_pending,
    Publishers.NovelMic.name: 			func_pending,
    Publishers.Mangagreat.name: 		func_pending,
    Publishers.Mangageko.name: 			func_pending,
    Publishers.Mangarolls.name: 		func_pending,
    Publishers.FirstKiss.name: 			func_pending,
}


async def scrape_switch(publisher: str, url: str) -> None:
    """
    Route scraping request to appropriate scraping function.

    Args:
        publisher: Name of the publisher
        url: URL to scrape
    """
    scrape_func = SCRAPE_FUNCTIONS.get(publisher, func_pending)
    await scrape_func(url)


async def async_scrape() -> None:
    """Asynchronously scrape all configured publisher URLs."""
    tasks = [scrape_switch(pub, url) for pub, url in publisher_url_pairs]
    await asyncio.gather(*tasks)


def scrapes() -> None:
    """
    Main entry point for comic scraping.

    Coordinates the scraping process across all publishers.
    """
    try:
        asyncio.run(async_scrape())
    finally:
        # Commit in memory changes to the database
        session.commit()
        # Close the database session
        session.close()
