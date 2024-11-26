from scrape.scrapper import scrape_url, check_chapter_extraction, register_comic
from helpers.logger import logger
from db import Types, Statuses, Publishers

log = logger(__name__)
publisher = Publishers.ManhuaPlus
default_comic_type = Types.Manhua
default_status = Statuses.OnAir


async def scrape_manhuaplus(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.find_all('div', attrs={'class': 'page-item-detail text'})
    check_chapter_extraction(chaps, publisher)
    title = 'Unknown'
    # Default comic type and status for publisher
    com_type = default_comic_type
    status = default_status
    for comic in chaps:
        try:
            # Locating cover
            cover = comic.div.a.img['src']
            # Internal div with title and chapters
            comic_int = comic.select("div.item-summary")[0]
            # Locating div used for title
            title = comic_int.select("div.post-title.font-title")[0].h3.a.text
            # Locating div used for chapters
            chap_int = comic_int.find_all(
                'div', attrs={'class': 'chapter-item'})
            # Locating the last chapter
            chap = chap_int[0].span.a.text
        except (ValueError, IndexError, KeyError, AttributeError) as error:
            log.error('scraping %s:%s %s', publisher.name, title, error)
            continue
        await register_comic(chap, title, com_type, cover, status, publisher)
