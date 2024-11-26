import cloudscraper
import json
import os
import re
import time
from bs4 import BeautifulSoup as beauty
from typing import Optional, List, Dict, Tuple
from db import Publishers, Types, Statuses, ComicDB
from db import save_comics_file, load_comics
from db.helpers import manage_multi_finds
from db.repo import comics_like_title
from scrape.url_switch import url_switch
from helpers.alert import add_alert_to_msg
from helpers.logger import logger

log = logger(__name__)
chaps_file = os.path.join(os.path.dirname(__file__), "../db/chaps.html")
TIME_OUT_LIMIT = 50
scraper = cloudscraper.create_scraper(browser='chrome')


async def scrape_url(url: str, str_to_file: str = ' ') -> beauty:
    # Make a GET request to the website
    try:
        with scraper.get(url, timeout=TIME_OUT_LIMIT) as response:
            if response.status_code != 200:
                log.warning('fetching %s server %s', url, response.status_code)
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
        log.warning('fetching %s timed out, %s %s', url, type(err), err)
        return beauty('', 'html.parser')


def check_chapter_extraction(chapters: list, publisher: Publishers) -> None:
    if len(chapters) < 1:
        log.error('%s needs function remake, no chaps found', str(publisher))


async def register_comic(
    chap: str,
    title: str,
    com_type: Types,
    cover: str,
    status: Statuses,
    publisher: Publishers,
    author: str = ''
) -> None:
    # Striping extra information from chapters like name and decimals
    (chap, title, cover) = strip_parameters(chap, title, cover, publisher)
    # Fetching from DB and JSON
    db_comics, session = comics_like_title(title)
    comics = [comic for comic in load_comics if title in comic["titles"]]
    # Check for multiple responses
    db_comics, title = manage_multi_finds(db_comics, com_type, title)
    if len(db_comics) == 0:
        log.info('%s Not Found in DB, creating new entry', title)
        db_comic_to_load = ComicDB(
          None, titles=title, current_chap=chap, cover=cover, com_type=com_type,
          status=status, published_in=publisher, author=author
        )
        session.add(db_comic_to_load)
        session.commit()

        db_comic_json_to_load = db_comic_to_load.toJSON()
        print('[  NEW   ]:', json.dumps(db_comic_json_to_load))
        load_comics.append(db_comic_json_to_load)
        save_comics_file(load_comics)
    if len(db_comics) != 1:
        log.warning('Abnormal length in db query: %s, [%s] impossible to parse',
                    len(db_comics), title)
        return
    # Check when fails fetching from JSON backup file
    if len(comics) == 0:
        comics = [comic for comic in load_comics if db_comics[0].id is comic["id"]]
        if len(comics) == 0:
            log.debug('%s - %s was not found in JSON, ID:%s',
                      publisher.name, title, db_comics[0].id)
            comics = [db_comics[0].toJSON()]
            load_comics.append(comics[0])
    # Updating publishers if new one found
    if publisher not in db_comics[0].get_published_in():
        db_comics[0].published_in += f'|{publisher}'
        comics[0]['published_in'].append(publisher)
        log.info('%s adding new publisher: %s', title, publisher)
    # Updating last chapter released
    if chap > db_comics[0].current_chap:
        db_comics[0].current_chap = chap
        comics[0]['current_chap'] = chap
        db_comics[0].last_update = int(time.time())
        comics[0]['last_update'] = int(time.time())
        if db_comics[0].track:
            add_alert_to_msg(title, chap, db_comics[0].get_published_in())
    # Update author
    if author != '' and db_comics[0].author == '':
        db_comics[0].author = author
        comics[0]['author'] = author
    # Update type
    if com_type != Types.Unknown and db_comics[0].com_type == Types.Unknown:
        db_comics[0].com_type = com_type
        comics[0]['com_type'] = com_type
    # Update status
    if status != Statuses.Unknown:
        db_comics[0].status = status
        comics[0]['status'] = status
    # Update cover for required publishers
    await update_cover_if_needed(db_comics, comics, cover, publisher)
    session.flush()
    save_comics_file(load_comics)


async def update_cover_if_needed(
    db_comics: list[ComicDB],
    comics: list[dict],
    cover: str,
    publisher: Publishers
) -> None:
    if cover == '' or db_comics[0].cover == cover:
        return
    # Update cover when no cover was present
    if not db_comics[0].cover:
        db_comics[0].cover = cover
        comics[0]['cover'] = cover
    # Update cover for ManhuaPlus and Reaper comics due to load restriction
    if (
        publisher != Publishers.ManhuaPlus and
        publisher != Publishers.ReaperScans and
        (
            Publishers.ManhuaPlus in db_comics[0].get_published_in() or
            Publishers.ReaperScans in db_comics[0].get_published_in()
        )
    ):
        db_comics[0].cover = cover
        comics[0]['cover'] = cover
    # Update url for asura, flame, manganato and realm
    if (
        publisher == Publishers.Asura or
        publisher == Publishers.FlameScans or
        publisher == Publishers.Manganato or
        publisher == Publishers.RealmScans
    ):
        db_comics[0].cover = cover
        comics[0]['cover'] = cover


def strip_parameters(
    chap: str,
    title: str,
    cover: str,
    publisher: Publishers
) -> tuple[int, str, str]:
    # Striping extra information from chapters like name and decimals
    new_chap = 0
    # Locating the chapter integer
    try:
        new_chap = int(re.findall(r'\d+', chap)[0])
    except (ValueError, IndexError) as error:
        log.error('%s, %s impossible to parse from %s', error, chap, title)
        return (new_chap, title, cover)
    # Locating cover url
    if cover.find('http') != -1:  # Replace cdn when found
        cover = cover[cover.find('http'):]
    elif cover.find('/') == 0:  # Add source url
        cover = url_switch.get(Publishers(publisher).name, [''])[0] + cover
    if len(cover) < 10:
        log.error('bad cover (%s) %s for comic %s', cover, publisher, title)
        cover = ''
    # Striping and capitalizing title for uniformity
    title = title.strip().capitalize()
    # Replace for novel comics syntax in LuminousScans
    title = title.replace('(novel)', ' - novel')
    return (new_chap, title, cover)


def com_status_parse(com_status_txt: str) -> Statuses:
    com_status_txt = com_status_txt.strip().lower()
    status_map = {
        'completed':  Statuses.Completed,
        'ongoing':    Statuses.OnAir,
        'hiatus':     Statuses.Break,
        'season end': Statuses.Break,
        'dropped':    Statuses.Dropped
    }
    return status_map.get(com_status_txt, Statuses.Unknown)


def com_type_parse(com_type_txt: str) -> Types:
    com_type = com_type_txt.strip().replace('NEW ', '').capitalize()
    try:
        com_type = Types[com_type]
    except (ValueError, IndexError, KeyError, AttributeError) as error:
        if str(error) == "'Comic'":
            com_type = Types["Manhwa"]
        elif str(error) == "'Hot'" or str(error) == "'Collab'":
            com_type = Types["Unknown"]
        else:
            com_type = Types["Unknown"]
            log.warning('[comic type] %s impossible to parse', error)
    return com_type
