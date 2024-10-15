from scrape.scrapper import scrape_url, check_chapter_extraction, register_comic
from db import Types, Statuses, Publishers

async def scrape_flame(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.find_all(class_="bsx")
    check_chapter_extraction(chaps, Publishers.FlameScans)
    for comic in chaps:
        # Default comic type for publisher
        com_type = Types.Manhwa
        # Locating cover
        cover = comic.a.div.img["src"]
        # Locating div used for title
        comic_int = comic.select("div.bigor")[0]
        title = comic_int.select("div.tt")[0].text
        # Locating div used chapter
        chap_int = comic_int.select("div.chapter-list")
        if len(chap_int) == 0:
            # These are the cases when a comic is portrayed as recommended
            continue
        chap = chap_int[0].a.div.div.text
        await register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.FlameScans)