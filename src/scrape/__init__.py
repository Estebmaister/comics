# scrape/__init__.py
## Scraping webs to save latest comics

import asyncio
import time, os, re, json
import cloudscraper
from bs4 import BeautifulSoup as beauty
from helpers.alert import add_alert_to_msg
from db import Types, Statuses, Publishers, ComicDB
from db import session, save_comics_file, load_comics
from db.helpers import manage_multi_finds
from db.repo import comics_like_title

scraper = cloudscraper.create_scraper(browser='chrome')
chaps_file = os.path.join(os.path.dirname(__file__), "../db/chaps.html")
TIME_OUT_LIMIT = 50
div_item_summary = "div.item-summary"

def strip_parameters(chap: str, title: str, cover: str):
    # Striping extra information from chapters like name and decimals
    try:
        chap = int(re.findall(r'\d+', chap)[0])
    except (ValueError, IndexError) as error:
        print(f'ERROR: {error}, {chap} impossible to parse from {title}')
        return
    # Replace cdn when found
    if '/cdn-cgi' in cover:
        # TODO: do not hardcode this
        cover = "https://realmscans.to" + cover
    cover = cover[cover.find("http"):]
    if len(cover) < 10:
        print(f'ERROR: bad cover {cover}')
        cover = ''
    # Striping and capitalizing title for uniformity
    title = title.strip().capitalize()
    # Replace for novel comics syntax in LuminousScans
    title = title.replace("(novel)", " - novel")
    return (chap, title, cover)

async def register_comic(chap: str, title: str, 
    com_type: Types, cover: str, status: Statuses, publisher: Publishers):
    (chap, title, cover) = strip_parameters(chap, title, cover)

    db_comics, session = comics_like_title(title)
    comics = [comic for comic in load_comics if title in comic["titles"]]
    ## Check for multiple responses
    db_comics, title = manage_multi_finds(db_comics, com_type, title)
    if len(db_comics) == 0:
        print(f'INFO: {title} Not Found in DB, creating new entry')
        db_comic_to_load = ComicDB(None, title, chap, cover, 
            int(time.time()), com_type, status, publisher)
        session.add(db_comic_to_load)
        session.commit()
        
        db_comic_json_to_load = db_comic_to_load.toJSON()
        print('NEW :', json.dumps(db_comic_json_to_load))
        load_comics.append(db_comic_json_to_load)
        save_comics_file(load_comics)
    
    elif len(db_comics) == 1:
        ## Check when fails fetching from JSON backup file
        if len(comics) == 0:
            print('WARN:', title, "wast'n found in JSON.", 
                    publisher, chap, cover)
            comics = [db_comics[0].toJSON()]
            load_comics.append(comics[0])

        ## Checking for more than one publisher
        if publisher not in db_comics[0].get_published_in():
            db_comics[0].published_in += f"|{publisher}"
            comics[0]["published_in"].append(publisher)
            print('INFO:', title, "adding new publisher:", publisher)

        ## Updating last chapter released
        if chap > db_comics[0].current_chap:
            db_comics[0].current_chap = chap
            comics[0]["current_chap"] = chap
            db_comics[0].last_update = int(time.time())
            comics[0]["last_update"] = int(time.time())
            if db_comics[0].track:
                add_alert_to_msg(title,chap,db_comics[0].get_published_in())
        
        ## Update cover for ManhuaPlus comics
        await update_cover_if_needed(db_comics, comics, cover, publisher, title)
        
        session.flush()
        save_comics_file(load_comics)
    else:
        print(f'WARN: Abnormal length in db query: {len(db_comics)}, '
            + f'[{title}] impossible to parse')

async def update_cover_if_needed(db_comics: [], comics: [], cover: str, 
    publisher: Publishers, title: str):
    if cover == '': return
    # Update cover for ManhuaPlus and Reaper comics due to load restriction
    if not db_comics[0].cover or (publisher != Publishers.ManhuaPlus and 
        publisher != Publishers.ReaperScans and
        (Publishers.ManhuaPlus in db_comics[0].get_published_in() or
            Publishers.ReaperScans in db_comics[0].get_published_in())):
        if db_comics[0].cover != cover:
            db_comics[0].cover = cover
            comics[0]['cover'] = cover
    # Update url for asura
    if (publisher == Publishers.Asura and db_comics[0].cover != cover) or (
        publisher == Publishers.FlameScans and db_comics[0].cover != cover):
        db_comics[0].cover = cover
        comics[0]['cover'] = cover
        print('INFO:', title, 'cover updated')

async def scrape(url: str, str_to_file: str = ' '):
    # Make a GET request to the website
    try:
        with scraper.get(url, timeout = TIME_OUT_LIMIT) as response:
            if response.status_code != 200:
                print(f'WARN: fetching {url} server {response.status_code}')
            # Parse the HTML content of the website
            soup = beauty(response.text, 'html.parser')
            # Printing scraped data
            if str_to_file in url:
                with open(chaps_file, 'w+') as file:
                    file.write(f'{soup}')
                    ## Writing only chapters divs
                    # divs = ' '.join( map( str,
                    #     soup.select('div.item__wrap')
                    # ))
                    # file.write(f'{divs}')
            return soup
    except Exception as err:
        print(f'WARN: fetching {url} timed out, {type(err)} {err}')
        return beauty('', 'html.parser')

def com_type_parse(com_type_txt: str):
    com_type = com_type_txt.replace('NEW ', '').capitalize()
    try:
        com_type = Types[com_type]
    except (ValueError, IndexError, KeyError, AttributeError) as error:
        if str(error) == "'Comic'":
            com_type = Types["Manhwa"]
        elif str(error) == "'Hot'" or str(error) == "'Collab'":
            com_type = Types["Unknown"]
        else:
            com_type = Types["Unknown"]
            print(f'WARN: Comic type {error} impossible to parse')
    return com_type

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
    soup = await scrape(url)
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
    soup = await scrape(url)
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

async def scrape_reset(url: str):
    print(f'WARN: {str(Publishers.ResetScans)} needs remake for scrape func')
    # await scrape_common_1(url, Publishers.ResetScans)

async def scrape_asura(url: str):
    await scrape_common_2(url, Publishers.Asura)
def scrape_publisher(publisher: Publishers, scrape_ver: int):
    if scrape_ver == 1:
        return lambda url: scrape_common_1(url, publisher)
    else:
        return lambda url: scrape_common_2(url, publisher)

async def scrape_flame(url: str):
    soup = await scrape(url)
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

async def scrape_nightscans(url: str):
    soup = await scrape(url)
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
    soup = await scrape(url)
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
    soup = await scrape(url)
    # Locating divs used for comics/novels
    chaps = soup.select("div.relative.flex.space-x-2.rounded.bg-white.p-2")
    check_chapter_extraction(chaps, Publishers.RealmScans)
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
    "https://asuracomics.com/"      :scrape_publisher(Publishers.Asura, 2),
    "https://void-scans.com/"       :scrape_publisher(Publishers.VoidScans, 2),
    "https://realmscans.to/"        :scrape_publisher(Publishers.RealmScans, 2),
    "https://luminousscans.com/"    :scrape_publisher(Publishers.LuminousScans, 2),
    "https://isekaiscan.com/"       :scrape_publisher(Publishers.IsekaiScan, 1),
    "https://en.leviatanscans.com"  :scrape_publisher(Publishers.LeviatanScans, 1),
    "https://nightscans.org/"       :scrape_nightscans,
    "https://reaperscans.com/"      :scrape_reaper,
    "https://manhuaplus.org/"       :scrape_manhuaplus,
    "https://flamecomics.com/"      :scrape_flame,
    "https://reset-scans.com/"      :scrape_reset,
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
