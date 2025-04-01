import math
from typing import List

from sqlalchemy.orm import Session as SessionType
from sqlalchemy.sql import text

from db import ComicDB, Session, Statuses, Types, load_comics, save_comics_file
from helpers.logger import logger

# Configure logging
log = logger(__name__)


class Pagination:
    def __init__(self, offset, limit, total_records, total_pages, current_page):
        self.offset = offset
        self.limit = limit
        self.total_records = total_records
        self.total_pages = total_pages
        self.current_page = current_page


def sql_check() -> any:
    with Session() as session:
        return session.execute(text('SELECT 1'))


def all_comics(_from: int = 0, limit: int = 20,
               only_tracked: bool = False, only_unchecked: bool = False,
               full_query: bool = False
               ) -> (List[ComicDB], Pagination):
    with Session() as session:
        if full_query:
            return session.query(ComicDB).all()

        partial_result = session.query(ComicDB).order_by(
            ComicDB.last_update.desc(), ComicDB.id
        )
    if only_tracked:
        if only_unchecked:
            partial_result = partial_result.filter(
                ComicDB.track == 1,
                ComicDB.current_chap != ComicDB.viewed_chap
            )
        else:
            partial_result = partial_result.filter(ComicDB.track == 1)
    total = partial_result.count()
    total_pages = math.ceil(total/limit)
    current_page = math.ceil(_from/limit + 1)
    pagination_data = Pagination(
        _from, limit, total, total_pages, current_page)
    return partial_result.offset(_from).limit(limit), pagination_data


def comic_by_id(id: int) -> (dict | None):
    '''
    >>> comic_by_id(10) -> Comic, session
    '''
    with Session() as session:
        comic = session.query(ComicDB).get(id)
        if comic is None:
            return None
        return comic.toJSON()


def delete_comic_by_id(id: int) -> (int):
    '''>>> delete_comic_by_id(10) -> rows_deleted: int'''
    with Session() as session:
        rows_deleted = session.query(ComicDB).filter(ComicDB.id == id).delete()
        if rows_deleted == 0:
            return rows_deleted
        try:
            json_comic = [com for com in load_comics if id == com["id"]][0]
            load_comics.remove(json_comic)
            save_comics_file(load_comics)
        except IndexError:
            log.debug('Comic ID %s not found in JSON backup', id)
        return rows_deleted


def update_comic_by_id(id: int, body: dict) -> (dict | None):
    '''>>> update_comic_by_id(10) -> jsonComic | None'''
    with Session() as session:
        comic = session.query(ComicDB).get(id)
        if comic is None:
            log.info('No comic found by ID %s', id)
            return None
        try:
            json_comic = [com for com in load_comics if id == com["id"]][0]
        except IndexError:
            log.debug('Comic ID %s not found in JSON backup, adding it', id)
            load_comics.append(comic.toJSON())
            json_comic = [com for com in load_comics if id == com["id"]][0]

        titles = body.get('titles')
        if titles is not None:
            comic.set_titles(titles)
            json_comic["titles"] = comic.get_titles()

        genres = body.get('genres')
        if genres is not None:
            genres = list(set([int(g) for g in body.get('genres', 0)]))
            comic.set_genres(genres)
            json_comic["genres"] = genres

        publishers = body.get('published_in')
        if publishers is not None:
            publishers = list(set([int(g) for g in body.get(
                'published_in', 0
            )]))
            comic.set_published_in(publishers)
            json_comic["published_in"] = publishers

        comic.author = body.get('author', comic.author)
        comic.cover = body.get('cover', comic.cover)
        comic.description = body.get('description', comic.description)
        comic.track = int(body.get('track', comic.track))
        comic.viewed_chap = int(body.get('viewed_chap', comic.viewed_chap))
        comic.current_chap = int(body.get('current_chap', comic.current_chap))
        comic.com_type = int(body.get('com_type', comic.com_type))
        comic.status = int(body.get('status', comic.status))
        comic.rating = int(body.get('rating', comic.rating))

        json_comic["author"] = comic.author
        json_comic["cover"] = comic.cover
        json_comic["description"] = comic.description
        json_comic["track"] = bool(comic.track)
        json_comic["viewed_chap"] = comic.viewed_chap
        json_comic["current_chap"] = comic.current_chap
        json_comic["com_type"] = Types(comic.com_type)
        json_comic["status"] = Statuses(comic.status)
        json_comic["rating"] = comic.rating

        comicJSON = comic.toJSON()
        session.commit()
        save_comics_file(load_comics)
        return comicJSON


def comics_like_title(title: str, session: SessionType) -> (List[ComicDB]):
    return session.query(ComicDB).filter(
        ComicDB.titles.like(f"%{title}%")
    ).order_by(
        ComicDB.last_update.desc(), ComicDB.id
    ).all()


def comics_by_title_no_case(
    title: str, _from: int = 0, limit: int = 20,
    only_tracked: bool = False, only_unchecked: bool = False,
    full_query: bool = False
) -> (List[ComicDB], Pagination):
    with Session() as session:
        partial_result = session.query(ComicDB).filter(
            ComicDB.titles.ilike(f"%{title.lower()}%")
        ).order_by(
            ComicDB.last_update.desc(), ComicDB.id
        )
    if only_tracked:
        if only_unchecked:
            partial_result = partial_result.filter(
                ComicDB.track == 1,
                ComicDB.current_chap != ComicDB.viewed_chap
            )
        else:
            partial_result = partial_result.filter(ComicDB.track == 1)
    if full_query:
        return partial_result.all()

    total = partial_result.count()
    total_pages = math.ceil(total/limit)
    current_page = math.ceil(_from/limit + 1)
    pagination_data = Pagination(
        _from, limit, total, total_pages, current_page)
    return partial_result.offset(_from).limit(limit), pagination_data


COMIC_NOT_FOUND = 'Comic {} not found'


def merge_comics(base_id: int, merging_id: int) -> (dict, str):
    '''>>> merge_comics(10, 24) -> ComicJSON: dict, None - error msg '''
    with Session() as session:
        comic = session.query(ComicDB).get(base_id)
        if comic is None:
            return None, COMIC_NOT_FOUND.format(base_id)
        d_comic = session.query(ComicDB).get(merging_id)
        if d_comic is None:
            return None, COMIC_NOT_FOUND.format(merging_id)
        if d_comic.com_type != 0 and comic.com_type != d_comic.com_type:
            return None, 'Comics to merge should be of the same type'
        try:
            json_comic = [
                com for com in load_comics if comic.id == com["id"]][0]
        except IndexError:
            load_comics.append(comic.toJSON())
            json_comic = [
                com for com in load_comics if comic.id == com["id"]][0]
        try:
            dj_comic = [
                com for com in load_comics if d_comic.id == com["id"]][0]
        except IndexError:
            load_comics.append(d_comic.toJSON())
            dj_comic = [
                com for com in load_comics if d_comic.id == com["id"]][0]

        titles = list(set(comic.get_titles() + d_comic.get_titles()))
        comic.set_titles(titles)
        genres = list(set(comic.get_genres() + d_comic.get_genres()))
        comic.set_genres(genres)
        publishers = list(
            set(comic.get_published_in() + d_comic.get_published_in()))
        comic.set_published_in(publishers)
        if comic.current_chap < d_comic.current_chap:
            comic.current_chap = d_comic.current_chap
        if comic.viewed_chap < d_comic.viewed_chap:
            comic.viewed_chap = d_comic.viewed_chap
        if not comic.track & d_comic.track:
            comic.track = d_comic.track
        if comic.rating == 0:
            comic.rating = d_comic.rating
        if comic.author == '':
            comic.author = d_comic.author
        if comic.description == '':
            comic.description = d_comic.description

        session.delete(d_comic)
        session.commit()
        comicJSON = comic.toJSON()
        session.close()

        json_comic["titles"] = titles
        json_comic["genres"] = genres
        json_comic["published_in"] = publishers
        json_comic["com_type"] = Types(comic.com_type)
        json_comic["viewed_chap"] = comic.viewed_chap
        json_comic["current_chap"] = comic.current_chap
        json_comic["track"] = bool(comic.track)
        json_comic["rating"] = comic.rating
        json_comic["author"] = comic.author
        json_comic["description"] = comic.description

        load_comics.remove(dj_comic)
        save_comics_file(load_comics)

        return comicJSON, None
