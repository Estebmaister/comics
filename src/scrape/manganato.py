from scrape.scrapper import scrape_url, check_chapter_extraction, register_comic
from helpers.logger import logger
from db import Types, Statuses, Publishers

log = logger(__name__)
publisher = Publishers.Manganato
default_comic_type = Types.Manhwa
default_status = Statuses.OnAir

async def scrape_manganato(url: str):
	soup = await scrape_url(url)
	# Locating divs used for comics
	chaps = soup.find_all('div', attrs = {'class':'content-homepage-item'}) 
	check_chapter_extraction(chaps, publisher)
	title = 'Unknown'
	for comic in chaps:
		# Default comic type and status for publisher
		com_type = default_comic_type
		status = default_status
		try:
			# Locating cover
			cover = comic.a.img['src']
			# Internal div with title and chapters
			comic_int = comic.select("div.content-homepage-item-right")[0]
			# Locating div used for title
			title = comic_int.h3.a.text
			# Locating div used for author
			author = comic_int.span.text
			# Locating div used for chapters
			chap_int = comic_int.find_all('p')
			# Locating the last chapter
			chap = chap_int[0].a.text
		except (ValueError, IndexError, KeyError, AttributeError) as error:
			log.error('scraping %s:%s %s', publisher.name, title, error)
			continue
		await register_comic(chap, title, com_type, cover, status, publisher, author)
