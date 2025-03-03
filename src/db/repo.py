import math
from typing import List

from sqlalchemy.sql import text

from db import ComicDB, Types, load_comics, save_comics_file, session


class Pagination:
    def __init__(self, offset, limit, total_records, total_pages, current_page):
        self.offset = offset
        self.limit = limit
        self.total_records = total_records
        self.total_pages = total_pages
        self.current_page = current_page


def sql_check() -> any:
    return session.execute(text('SELECT 1'))


def all_comics(_from: int = 0, limit: int = 20,
               only_tracked: bool = False, only_unchecked: bool = False,
               full_query: bool = False
               ) -> (List[ComicDB], Pagination):
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


def comic_by_id(id: int) -> (ComicDB, any):
    return session.query(ComicDB).get(id), session


def comics_like_title(title: str) -> (List[ComicDB], any):
    return session.query(ComicDB).filter(
        ComicDB.titles.like(f"%{title}%")
    ).order_by(
        ComicDB.last_update.desc(), ComicDB.id
    ).all(), session


def comics_by_title_no_case(
    title: str, _from: int = 0, limit: int = 20,
    only_tracked: bool = False, only_unchecked: bool = False,
    full_query: bool = False
) -> (List[ComicDB], Pagination):
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
    '''>>> merge_comics(10, 24) -> Comic.toJSON(), None - error msg '''
    comic, session = comic_by_id(base_id)
    if comic is None:
        return None, COMIC_NOT_FOUND.format(base_id)
    d_comic = session.query(ComicDB).get(merging_id)
    if d_comic is None:
        return None, COMIC_NOT_FOUND.format(merging_id)
    if d_comic.com_type != 0 and comic.com_type != d_comic.com_type:
        return None, 'Comics to merge should be of the same type'
    try:
        json_comic = [com for com in load_comics if comic.id == com["id"]][0]
    except IndexError:
        load_comics.append(comic.toJSON())
        json_comic = [com for com in load_comics if comic.id == com["id"]][0]
    try:
        dj_comic = [com for com in load_comics if d_comic.id == com["id"]][0]
    except IndexError:
        load_comics.append(d_comic.toJSON())
        dj_comic = [com for com in load_comics if d_comic.id == com["id"]][0]

    titles = list(set(comic.get_titles() + d_comic.get_titles()))
    comic.set_titles(titles)
    genres = list(set(comic.get_genres() + d_comic.get_genres()))
    comic.set_genres(genres)
    publishers = list(
        set(comic.get_published_in() + d_comic.get_published_in()))
    comic.set_published_in(publishers)
    if comic.current_chap < d_comic.current_chap:
        comic.current_chap = d_comic.current_chap

    json_comic["titles"] = titles
    json_comic["genres"] = genres
    json_comic["com_type"] = Types(comic.com_type)
    json_comic["published_in"] = publishers

    session.delete(d_comic)
    session.commit()
    load_comics.remove(dj_comic)
    save_comics_file(load_comics)
    return comic.toJSON(), None
