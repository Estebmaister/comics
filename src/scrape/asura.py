from scrape.scrapper import scrape_url, check_chapter_extraction, register_comic
from helpers.logger import logger
from db import Types, Statuses, Publishers

log = logger(__name__)
publisher = Publishers.Asura
default_comic_type = Types.Manhwa
default_status = Statuses.OnAir


async def scrape_asura(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.find_all(
        'div', attrs={'class': 'grid grid-rows-1 grid-cols-12 m-2'})
    check_chapter_extraction(chaps, publisher)
    title = 'Unknown'
    # Default comic type for publisher
    com_type = default_comic_type
    status = default_status
    for comic in chaps:
        try:
            # Locating cover
            cover = comic.div.div.a.img['src']
            # Internal div with title and chapters
            comic_int = comic.select('div')[2]
            # Locating div used for title
            title = comic_int.span.a.text
            title = title.replace('...', '')
            # Locating div used for chapters
            chap_int = comic_int.find_all('span')
            if len(chap_int) == 0:
                # These are the cases when a comic is portrayed as recommended
                continue
            # Locating the chapter
            chap = chap_int[1].div.div.a.span.div.p.text
        except (ValueError, IndexError, KeyError, AttributeError) as error:
            log.error('scraping %s:%s %s', publisher.name, title, error)
            continue
        await register_comic(chap, title, com_type, cover, status, publisher)
