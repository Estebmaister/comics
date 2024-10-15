# scrape/__init__.py
## Scraping webs to save latest comics

import asyncio
from db import Types, Statuses, Publishers
from db import session
from scrape.scrapper import scrape_url, register_comic, com_type_parse
from scrape.asura import scrape_asura
from scrape.flame import scrape_flame

div_item_summary = "div.item-summary"

def check_chapter_extraction(chapters: [], publisher: Publishers):
    if len(chapters) < 1: 
        print(f'WARN: {str(publisher)} needs remake for scrape func')

async def scrape_chapter(comic, int_path: str, title_path: str, chap_path: str):
    # Locating div used for title and chapter
    comic_int = comic.select(int_path)[0]
    try:
        title = comic_int.select(title_path)[0].h3.a.text.strip()
    except (ValueError, IndexError, KeyError, AttributeError) as err:
        raise AttributeError(f'-Title- impossible to parse {err} html: {comic_int}')
    chap = 0
    try:
        chap = comic_int.select(chap_path)[0].a.text
    except (ValueError, IndexError, KeyError, AttributeError) as err:
        raise AttributeError(f'-Chapter- impossible to parse for {title} {err}')
    # Locating and parsing comic type
    com_type = com_type_parse("Unknown")
    try:
        com_type = com_type_parse(comic.div.a.span.text)
    except AttributeError:
        pass
    # Locating cover
    cover = ''
    try:
        cover = comic.div.a.img["data-src"]
    except KeyError:
        cover = comic.div.a.img["src"]
    return chap, title, com_type, cover

async def scrape_common_1(url: str, publisher: Publishers):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.select("div.page-item-detail.manga")
    check_chapter_extraction(chaps, publisher)
    for comic in chaps:
        try:
            chap, title, com_type, cover = await scrape_chapter(
                    comic, div_item_summary,
                    "div.post-title.font-title", 
                    "span.chapter.font-meta"
                )
            await register_comic(chap, title, com_type, cover,
                Statuses.OnAir, publisher)
        except (ValueError, IndexError, KeyError, AttributeError) as error:
            print(f'ERROR: scraping {str(publisher)}:{title} {error}')
            continue


async def scrape_common_2(url: str, publisher: Publishers):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.find_all(class_="uta")
    check_chapter_extraction(chaps, publisher)
    for comic in chaps:
        # Locating div used for title and chapter
        comic_int = comic.find(class_="luf")
        title = comic_int.h4.text.strip()
        chap = '0'
        if (comic_int.li):
            chap = comic_int.li.a.text
        else:
            # Normal case for VoidScans with upcoming comics
            if publisher != Publishers.VoidScans:
                print(f'ERROR parsing -Chapter-  for {title}'+
                    f'from {str(publisher)}, html:{comic_int}')
        # Locating comic type
        if (comic.ul):
            com_type = com_type_parse(comic.ul["class"][0])
        else:
            com_type = com_type_parse('Unknown')
            # Normal case for VoidScans
            if publisher != Publishers.VoidScans:
                print(f'ERROR parsing -Comic type- for {title}'+
                    f'from {str(publisher)}, html:{comic}')
        # Locating cover
        cover = ''
        try:
            cover = comic.div.a.img["src"]
        except KeyError:
            try:
                cover = comic.div.a.img["data-cfsrc"]
            except KeyError:
                print(f'ERROR parsing -Cover- for {title} '+
                    f'from {str(publisher)}, html:{comic.div.a.img}'
                )
        await register_comic(chap, title, com_type, cover,
            Statuses.OnAir, publisher)

def scrape_publisher(publisher: Publishers, scrape_ver: int):
    if scrape_ver == 1:
        return lambda url: scrape_common_1(url, publisher)
    else:
        return lambda url: scrape_common_2(url, publisher)

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
        
async def scrape_manhuaplus(url: str):
    soup = await scrape_url(url)
    # Locating divs used for comics
    chaps = soup.select("div.col-6.col-md-3.badge-pos-2")
    check_chapter_extraction(chaps, Publishers.ManhuaPlus)
    for comic in chaps:
        # Default comic type for publisher
        com_type = Types.Manhua
        # Locating cover
        cover = comic.div.div.a.img["data-src"]
        # Locating div used for title and chapter
        comic_int = comic.select(div_item_summary)[0]
        title = comic_int.div.h3.a.text
        chap = comic_int.select("div.list-chapter")[0].div.span.a.text
        await register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.ManhuaPlus)

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
                print( f'ERROR: scraping {str(Publishers.ReaperScans)}: ' +
                f'-Cover- for {title}, html: [{comic.div.a.img}]')
        
        await register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.ReaperScans)


async def func_pending(url: str):
    pass # await print(url, "not implemented") #TODO

url_switch = {
    # "https://nightsup.net/"         :scrape_nightscans,
    # "https://manhuaplus.org/"       :scrape_manhuaplus,
    # "https://reaperscans.com/"      :scrape_reaper,
    "https://asuracomic.net/"       :scrape_asura,
    "https://flamecomics.com/"      :scrape_flame,
    "https://void-scans.com/"       :scrape_publisher(Publishers.VoidScans, 2),
    "https://rizzfables.com/"       :scrape_publisher(Publishers.RealmScans, 2),
    "https://en.leviatanscans.com"  :scrape_publisher(Publishers.LeviatanScans, 1),
    "https://luminousscans.com/"    :func_pending,
    "https://isekaiscan.com/"       :func_pending,
    "https://reset-scans.com/"      :func_pending,
    "https://drakescans.com/"       :func_pending,
    "https://novelmic.com/"         :func_pending,
    "https://mangagreat.com/"       :func_pending,
    "https://mangageko.com/"        :func_pending,
    "https://mangarolls.com/rolls"  :func_pending,
    "https://manganato.com/"        :func_pending,
    "https://1stkissmanga.me/"      :func_pending,
}
async def scrape_switch(url: str):
    return await url_switch.get(url, func_pending)(url)
async def async_scrape():
    # await scrape("https://en.leviatanscans.com/", 'levi')
    await asyncio.gather(*[scrape_switch(url) for url in url_switch.keys()])
def scrapes():
    asyncio.run(async_scrape())
    session.commit()
