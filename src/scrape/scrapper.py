import cloudscraper
import os
from db import Publishers
from bs4 import BeautifulSoup as beauty
import time, re, json
from helpers.alert import add_alert_to_msg
from db import Types, Statuses, Publishers, ComicDB
from db import save_comics_file, load_comics
from db.helpers import manage_multi_finds
from db.repo import comics_like_title

chaps_file = os.path.join(os.path.dirname(__file__), "../db/chaps.html")
TIME_OUT_LIMIT = 50
scraper = cloudscraper.create_scraper(browser='chrome')

async def scrape_url(url: str, str_to_file: str = ' '):
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
                    file.write(f'{soup.prettify()}')
                    # Writing only chapters divs
                    # divs = ' '.join( map( str,
                    #     soup.select('div.item__wrap')
                    # ))
                    # file.write(f'{divs}')
            return soup
    except Exception as err:
        print(f'WARN: fetching {url} timed out, {type(err)} {err}')
        return beauty('', 'html.parser')

def check_chapter_extraction(chapters: [], publisher: Publishers): # type: ignore
    if len(chapters) < 1: 
        print(f'WARN: {str(publisher)} needs remake for scrape func')


async def register_comic(
        chap: str, 
        title: str, 
        com_type: Types, 
        cover: str, 
        status: Statuses, 
        publisher: Publishers
    ):
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

async def update_cover_if_needed(db_comics: [], comics: [], cover: str,  # type: ignore
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

