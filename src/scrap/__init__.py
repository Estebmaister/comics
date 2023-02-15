# scrap/__init__.py
## Scraping webs to show latest comics

import time, os, re, json
import cloudscraper
from bs4 import BeautifulSoup as beauty
from helpers.alert import add_alert_to_msg
from db import Types, Statuses, Publishers, ComicDB, save_comics_file, load_comics
from db.helpers import manage_multi_finds
from db.repo import comics_by_title

scraper = cloudscraper.create_scraper(browser='chrome')
urls = {"asura":    "https://www.asurascans.com/",
    "reaper":       "https://reaperscans.com/",
    "manhuaplus":   "https://manhuaplus.com/",
    "flamescans":   "https://flamescans.org/",
    "luminousscans":"https://luminousscans.com/",
    "resetscans":   "https://reset-scans.com/",
    "isekaiscan":   "https://isekaiscan.com/",
    "realmscans":  "https://realmscans.com/",
    "nightscans":  "https://nightscans.org/",
    "voidscans":   "https://void-scans.org/",
    "drakescans":  "https://drakescans.com/",
    "novelmic":    "https://novelmic.com/",
    "mangagreat":  "https://mangagreat.com/",
    "mangageko":   "https://mangageko.com/",
    "mangarolls":  "https://mangarolls.com/rolls/",
    "manganato":   "https://manganato.com/"}

chaps_file = os.path.join(os.path.dirname(__file__), "../db/chaps.html")

div_item_summary = "div.item-summary"

def strip_parameters(chap, title, cover):
    # Striping extra information from chapters like name and decimals
    try:
        chap = int(re.findall(r'\d+', chap)[0])
    except (ValueError, IndexError) as error:
        print(f'{error}, {chap} impossible to parse from {title}')
        return
    # Replace cdn when encountered
    cover = cover[cover.find("http"):]
    # Striping and capitalizing title for uniformity
    title = title.strip().capitalize()
    # Replace for novel comics syntax in LuminousScans
    title = title.replace("(novel)", " - novel")
    return (chap, title, cover)

def register_comic(chap: str, title: str, 
    com_type: Types, cover: str, status: Statuses, publisher: Publishers):
    (chap, title, cover) = strip_parameters(chap, title, cover)

    db_comics, session = comics_by_title(title)
    comics = [comic for comic in load_comics if title in comic["titles"]]
    ## Check for multiple responses
    db_comics, title = manage_multi_finds(db_comics, com_type, title)
    if len(db_comics) == 0:
        print(f'{title} Not Found in DB, creating new entry')
        db_comic_to_load = ComicDB(None, title, chap, cover, 
            int(time.time()), com_type, status, publisher)
        
        session.add(db_comic_to_load)
        session.commit()
        load_comics.append(db_comic_to_load.toJSON())
        save_comics_file(load_comics)
        
        print(json.dumps(db_comic_to_load.toJSON()))
    elif len(db_comics) == 1:
        ## Checking for more than one publisher
        if publisher not in db_comics[0].get_published_in():
            db_comics[0].published_in += f"|{publisher}"
            comics[0]["published_in"].append(publisher)
            print(title, "adding new publisher:", publisher)

        ## Updating last chapter released
        if chap > db_comics[0].current_chap:
            db_comics[0].current_chap = chap
            comics[0]["current_chap"] = chap
            db_comics[0].last_update = int(time.time())
            comics[0]["last_update"] = int(time.time())
            if db_comics[0].track:
                add_alert_to_msg(title,chap,db_comics[0].get_published_in())
        
        ## Update cover for ManhuaPlus comics
        update_cover_manhuaplus(db_comics, comics, cover, publisher, title)
        
        session.commit()
        save_comics_file(load_comics)
    else:
        print(f'Abnormal length in db query: {len(db_comics)}, '
            + f'[{title}] impossible to parse')

def update_cover_manhuaplus(db_comics, comics, cover, publisher, title):
    '''Update cover for ManhuaPlus comics due to load restriction'''
    if not db_comics[0].cover or (publisher != Publishers.ManhuaPlus and 
        Publishers.ManhuaPlus in db_comics[0].get_published_in()):
        if db_comics[0].cover != cover:
            db_comics[0].cover = cover
            comics[0]["cover"] = cover
            print(title, "cover updated")

def scrap(url: str, str_to_file: str = ' '):
    # Make a GET request to the website
    response = scraper.get(url)
    # Parse the HTML content of the website
    soup = beauty(response.text, "html.parser")
    # Closing connection
    scraper.close()
    # Printing scraped data
    with open(chaps_file, "w+") as file:
        if str_to_file in url:
            file.write(f'{soup}')
            ## Writing only chapters divs
            # divs = " ".join( map( str,
            #     soup.select("div.item__wrap")
            # ))
            # file.write(f'{divs}')
    return soup

def scrap_luminousscans():
    soup = scrap(urls["luminousscans"])
    # Locating divs used for comics
    chaps = soup.find_all(class_="uta")
    for comic in chaps:
        # Locating comic type
        com_type = com_type_parse(comic.ul["class"][0])
        # Locating cover
        cover = comic.div.a.img["src"]
        # Locating div used for title and chapter
        comic_int = comic.select("div.luf")[0]
        title = comic_int.a.h4.text
        chap = comic_int.li.a.text

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.LuminousScans)

def scrap_flamescans():
    soup = scrap(urls["flamescans"])
    # Locating divs used for comics
    chaps = soup.find_all(class_="bsx")
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
            continue
        chap = chap_int[0].a.div.div.text

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.FlameScans)

def scrap_manhuaplus():
    soup = scrap(urls["manhuaplus"])
    # Locating divs used for comics
    chaps = soup.select("div.col-6.col-md-3.badge-pos-2")
    for comic in chaps:
        # Default comic type for publisher
        com_type = Types.Manhua
        # Locating cover
        cover = comic.div.div.a.img["data-src"]
        # Locating div used for title and chapter
        comic_int = comic.select(div_item_summary)[0]
        title = comic_int.div.h3.a.text
        chap = comic_int.select("div.list-chapter")[0].div.span.a.text

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.ManhuaPlus)

def scrap_reaper():
    soup = scrap(urls["reaper"])
    # Locating divs used for comics/novels
    chaps = soup.select("div.relative.flex.space-x-2.rounded.bg-white.p-2")
    # Flag to separate comics and novels
    comics_per_page = 8
    for comic in chaps:
        # Default comic type for publisher
        com_type = Types.Manhwa
        comics_per_page -= 1
        if comics_per_page < 0:
            com_type = Types.Novel
        # Locating cover
        cover = comic.div.a.img["src"]
        # Locating div used for title and chapter
        comic_int = comic.select("div.min-w-0.flex-1")[0]
        title = comic_int.div.p.a.text
        chap = comic_int.div.div.a.text

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.ReaperScans)

def scrap_asura():
    soup = scrap(urls["asura"])
    # Locating divs used for comics
    chaps = soup.find_all(class_="uta")
    for comic in chaps:
        # Locating comic type
        com_type = com_type_parse(comic.ul["class"][0])
        # Locating cover
        cover = comic.div.a.img["src"]
        # Locating div used for title and chapter
        comic_int = comic.find(class_="luf")
        title = comic_int.h4.text
        chap = comic_int.li.a.text

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.Asura)

        title = title[:27] + '...' if len(title) > 30 else '{:30}'.format(title)
        # print(f"{title} ch {chap}")

def scrap_realmscans():
    soup = scrap(urls["realmscans"])
    # Locating divs used for comics
    chaps = soup.find_all(class_="uta")
    for comic in chaps:
        # Locating and parsing comic type
        com_type = com_type_parse(comic.ul["class"][0])
        # Locating cover
        cover = comic.div.a.img["src"]
        # Locating div used for title and chapter
        comic_int = comic.find(class_="luf")
        title = comic_int.h4.text
        chap = comic_int.li.a.text

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.RealmScans)

def com_type_parse(com_type_txt: str):
    com_type = com_type_txt.replace("NEW ", "").capitalize()
    try:
        com_type = Types[com_type]
    except KeyError as ke:
        if str(ke) == "'Comic'":
            com_type = Types["Manhwa"]
        else:
            com_type = Types["Unknown"]
    except AttributeError:
            com_type = Types["Unknown"]
    return com_type

def scrap_chapter(comic, int_path: str, title_path: str, chap_path: str):
    # Locating and parsing comic type
    com_type = com_type_parse(comic.div.a.span.text)
    # Locating cover
    cover = comic.div.a.img["data-src"]
    # Locating div used for title and chapter
    comic_int = comic.select(int_path)[0]
    title = comic_int.select(title_path)[0].h3.a.text
    chap = comic_int.select(chap_path)[0].a.text

    return chap, title, com_type, cover

def scrap_resetscans():
    soup = scrap(urls["resetscans"])
    # Locating divs used for comics
    chaps = soup.select("div.page-item-detail.manga")
    for comic in chaps:
        chap, title, com_type, cover = scrap_chapter(comic, div_item_summary,
            "div.post-title.font-title", "span.chapter.font-meta")

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.ResetScans)

def scrap_isekaiscan():
    soup = scrap(urls["isekaiscan"])
    # Locating divs used for comics
    chaps = soup.select("div.page-item-detail.manga")
    for comic in chaps:
        chap, title, com_type, cover = scrap_chapter(comic, div_item_summary,
            "div.post-title.font-title", "span.chapter.font-meta")

        register_comic(chap, title, com_type, cover,
            Statuses.OnAir, Publishers.IsekaiScan)
        # print(f"{title} ch {chap}, {com_type}, {cover}")

def scraps():
    scrap_manhuaplus()
    scrap_asura()
    scrap_reaper()
    scrap_flamescans()
    scrap_luminousscans()
    scrap_resetscans()
    scrap_isekaiscan()
    scrap_realmscans()
    # scrap(urls["isekaiscan"], "ise")
