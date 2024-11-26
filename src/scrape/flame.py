from scrape.scrapper import scrape_url, check_chapter_extraction, register_comic
from helpers.logger import logger
from db import Types, Statuses, Publishers

log = logger(__name__)
publisher = Publishers.FlameScans
default_comic_type = Types.Manhwa
default_status = Statuses.OnAir


async def scrape_flame(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.find_all(class_='bsx')
    check_chapter_extraction(chaps, publisher)
    title = 'Unknown'
    # Default comic type for publisher
    com_type = default_comic_type
    status = default_status
    for comic in chaps:
        try:
            # Locating cover
            cover = comic.a.div.img['src']
            # Locating div used for title
            comic_int = comic.select('div.bigor')[0]
            title = comic_int.select('div.tt')[0].text
            # Locating div used chapter
            chap_int = comic_int.select('div.chapter-list')
            if len(chap_int) == 0:
                # These are the cases when a comic is portrayed as recommended
                continue
            chap = chap_int[0].a.div.div.text
        except (ValueError, IndexError, KeyError, AttributeError) as error:
            log.error('scraping %s:%s %s', publisher.name, title, error)
            continue
        await register_comic(chap, title, com_type, cover, status, publisher)
