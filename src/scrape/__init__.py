# scrape/__init__.py
# Scraping webs to save latest comics

import asyncio
from db import Types, Statuses, Publishers
from db import session
from scrape.scrapper import scrape_url, register_comic, com_type_parse
from scrape.asura import scrape_asura
from scrape.flame import scrape_flame
from scrape.manhuaplus import scrape_manhuaplus
from scrape.manganato import scrape_manganato
from scrape.realm import scrape_realm
from helpers.logger import logger
from scrape.url_switch import publisher_url_pairs

log = logger(__name__)
div_item_summary = "div.item-summary"


def check_chapter_extraction(chapters: [], publisher: Publishers):
    # deprecated
    if len(chapters) < 1:
        log.warning('%s needs remake for scrape func', str(publisher))


async def scrape_nightscans(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.find_all(class_="bsx")
    check_chapter_extraction(chaps, Publishers.NightScans)
    for comic in chaps:
        # Default comic type for publisher
        com_type = com_type_parse(comic.select("a")[0].div.span["class"][1])
        # Locating cover
        cover = comic.a.div.img["src"]
        # Locating div used for title
        comic_int = comic.select("div.bigor")[0]
        title = comic_int.select("div.tt")[0].text
        # Locating div used chapter
        chap_int = comic_int.find_all('li')
        if len(chap_int) == 0:
            # These are the cases when a comic is portrayed as recommended
            continue
        chap = chap_int[0].span.a.text
        await register_comic(chap, title, com_type, cover,
                             Statuses.OnAir, Publishers.NightScans)


async def scrape_reaper(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics/novels
    chaps = soup.select("div.relative.flex.space-x-2.rounded.bg-white.p-2")
    check_chapter_extraction(chaps, Publishers.ReaperScans)
    # Flag to separate comics and novels
    comics_per_page = 8
    for comic in chaps:
        # Default comic type for publisher
        com_type = Types.Manhwa
        comics_per_page -= 1
        if comics_per_page < 0:
            com_type = Types.Novel
        # Locating div used for title and chapter
        comic_int = comic.select("div.min-w-0.flex-1")[0]
        title = comic_int.div.p.a.text.strip()
        chap = comic_int.div.div.a.text
        # Locating cover
        cover = ''
        try:
            cover = comic.div.a.img["src"]
        except KeyError:
            try:
                cover = comic.div.a.img["data-cfsrc"]
            except KeyError:
                print(f'ERROR: scraping {str(Publishers.ReaperScans)}: ' +
                      f'-Cover- for {title}, html: [{comic.div.a.img}]')

        await register_comic(chap, title, com_type, cover,
                             Statuses.OnAir, Publishers.ReaperScans)


async def func_pending(url: str):
    # Function used for sites that are not yet implemented
    pass


async def site_closed(url: str):
    # Function used for sites that were closed
    pass

scrape_func_switch = {
    # Publishers(Publishers.NightScans).name: 		scrape_nightscans,  # TODO
    # Publishers(Publishers.ReaperScans).name: 		scrape_reaper,  # TODO
    Publishers(Publishers.ManhuaPlus).name: 		scrape_manhuaplus,
    Publishers(Publishers.Asura).name: 					scrape_asura,
    Publishers(Publishers.FlameScans).name: 		scrape_flame,
    Publishers(Publishers.RealmScans).name: 		scrape_realm,
    Publishers(Publishers.Manganato).name: 			scrape_manganato,
    Publishers(Publishers.LeviatanScans).name: 	site_closed,
    Publishers(Publishers.LuminousScans).name: 	func_pending,
    Publishers(Publishers.IsekaiScan).name: 		func_pending,
    Publishers(Publishers.VoidScans).name: 			func_pending,
    Publishers(Publishers.ResetScans).name: 		func_pending,
    Publishers(Publishers.DrakeScans).name: 		func_pending,
    Publishers(Publishers.NovelMic).name: 			func_pending,
    Publishers(Publishers.Mangagreat).name: 		func_pending,
    Publishers(Publishers.Mangageko).name: 			func_pending,
    Publishers(Publishers.Mangarolls).name: 		func_pending,
    Publishers(Publishers.FirstKiss).name: 			func_pending,
}


async def scrape_switch(pub: str, url: str):
    return await scrape_func_switch.get(pub, func_pending)(url)


async def async_scrape():
    await asyncio.gather(*[scrape_switch(pub, url) for pub, url in publisher_url_pairs])


def scrapes():
    asyncio.run(async_scrape())
    session.commit()
    session.close()
