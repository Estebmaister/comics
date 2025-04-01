"""
Comic scraping package initialization.

This module provides functions to scrape comics from various publisher websites.
It coordinates the scraping process across different publishers and manages the scraping workflow.
"""

import asyncio
from typing import Callable, Dict

from sqlalchemy.orm import Session as SessionType

from db import Publishers, Session
from helpers.alert import send_reminder
from helpers.logger import logger
from scrape.asura import scrape_asura
from scrape.demonic import scrape_demonic
from scrape.flame import scrape_flame
from scrape.manganato import scrape_manganato
from scrape.manhuaplus import scrape_manhuaplus
from scrape.realm import scrape_realm
from scrape.url_switch import publisher_url_pairs

# Configure logging
log = logger(__name__)


def scrapes() -> None:
    """Main synchronous entry point for comics scraping."""
    asyncio.run(async_scrape_wrapper())


async def async_scrape_wrapper():
    """Wrapper to manage the entire comics scraping process asynchronous."""
    with Session() as session:
        try:
            await async_scrape(session)
            session.commit()
        except Exception as e:
            session.rollback()
            log.error(f'Scraping error: {e}')
        finally:
            session.close()
            send_reminder()
            log.info('Scraping completed')


async def async_scrape(session: SessionType) -> None:
    """Asynchronously scrape all configured publisher URLs."""
    tasks = [scrape_switch(pub, url, session)
             for pub, url in publisher_url_pairs]
    await asyncio.gather(*tasks)


async def scrape_switch(publisher: str, url: str, session: SessionType) -> None:
    """Route scraping request to appropriate scraping function."""
    scrape_func = SCRAPE_FUNCTIONS.get(publisher, func_pending)
    await scrape_func(url, session)


async def func_pending(url: str, session: SessionType) -> None:
    """Placeholder for pending scraper implementation."""
    pass


async def site_closed(url: str, session: SessionType) -> None:
    """Handler for closed/inactive sites."""
    pass

# Map publisher names to their scraping functions
SCRAPE_FUNCTIONS: Dict[str, Callable] = {
    # Publishers.NightScans.name: 		scrape_nightscans,  # TODO: outdated
    # Publishers.ReaperScans.name: 		scrape_reaper,  # TODO: outdated
    Publishers.ManhuaPlus.name: 		scrape_manhuaplus,
    Publishers.Asura.name: 				scrape_asura,
    Publishers.FlameScans.name: 		scrape_flame,
    Publishers.RealmScans.name: 		scrape_realm,
    Publishers.DemonicScans.name: 	    scrape_demonic,
    Publishers.Manganato.name: 			site_closed,
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
