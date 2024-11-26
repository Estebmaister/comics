from scrape.scrapper import scrape_url, check_chapter_extraction, register_comic
from scrape.scrapper import com_type_parse, com_status_parse
from helpers.logger import logger
from db import Types, Statuses, Publishers

log = logger(__name__)
publisher = Publishers.RealmScans
default_comic_type = Types.Manhwa
default_status = Statuses.OnAir

async def scrape_realm(url: str):
  soup = await scrape_url(url)
  # Locating divs used for comics
  chaps = soup.find_all(class_='uta')
  check_chapter_extraction(chaps, publisher)
  title = 'Unknown'
  for comic in chaps:
    try:
      # Locating cover
      cover = comic.div.a.img['src']
      # Locating div used for status
      status = com_status_parse(comic.div.a.div.span.text)
      # Locating div used for title
      comic_int = comic.select('div.luf')[0]
      title = comic_int.a.h4.text.strip()
      # Locating div used chapter
      chap_int = comic_int.select('ul')
      if len(chap_int) == 0:
        # These are the cases when a comic is portrayed as recommended
        continue
      com_type = com_type_parse(chap_int[0]['class'][0])
      chap = chap_int[0].li.a.text.strip()
    except (ValueError, IndexError, KeyError, AttributeError) as error:
      log.error('scraping %s:%s %s', publisher.name, title, error)
      continue
    await register_comic(chap, title, com_type, cover, status, publisher)