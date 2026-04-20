import json
import math
from typing import Iterable, List, Sequence

from sqlalchemy.orm import Session as SessionType
from sqlalchemy.sql import text
from sqlalchemy.exc import IntegrityError, SQLAlchemyError

from db import ComicDB, Session, Statuses, Types, load_comics, save_comics_file
from db.identity import (build_identity_key, build_identity_key_from_titles,
                         merge_unique_values, normalize_title_variants)
from helpers.text import normalize_text
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
    '''List all comics with pagination, optional filters
    >>> all_comics(0, 20, False, False, False) -> (List[ComicDB], Pagination)
    '''
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
    if limit < 1:  # limit must be at least 1
        log.warning("Invalid pagination parameters - limit: %s", limit)
        limit = 1
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
        finally:
            session.commit()
        return rows_deleted


def create_comic(comic: ComicDB, session: SessionType = None) -> (dict | None):
    if session is None:
        try:
            with Session() as s:
                s.add(comic)
                s.commit()
                comicJSON = comic.toJSON()
            load_comics.append(comicJSON)
            save_comics_file(load_comics)
        except Exception as err:
            log.error('Failed to create comic: %s', err)
            return None
    else:
        session.add(comic)
        session.flush()
        comicJSON = comic.toJSON()
        load_comics.append(comicJSON)

    log.info('Created new entry: %s', json.dumps(comicJSON))
    return comicJSON


def rebuild_json_backup_from_db(
    session: SessionType = None,
    *,
    persist_file: bool = True,
) -> List[dict]:
    if session is None:
        with Session() as current_session:
            comics = current_session.query(ComicDB).order_by(ComicDB.id).all()
    else:
        comics = session.query(ComicDB).order_by(ComicDB.id).all()

    load_comics[:] = [comic.toJSON() for comic in comics]
    if persist_file:
        save_comics_file(load_comics)
    return load_comics


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
        comic.normalize_titles()

        json_comic["author"] = comic.author
        json_comic["cover"] = comic.cover
        json_comic["description"] = comic.description
        json_comic["track"] = bool(comic.track)
        json_comic["viewed_chap"] = comic.viewed_chap
        json_comic["current_chap"] = comic.current_chap
        json_comic["titles"] = comic.get_titles()
        json_comic["com_type"] = Types(comic.com_type)
        json_comic["status"] = Statuses(comic.status)
        json_comic["rating"] = comic.rating

        comicJSON = comic.toJSON()
        session.commit()
        save_comics_file(load_comics)
        return comicJSON


def comics_like_title(title: str, session: SessionType = None) -> (List[ComicDB]):
    title = normalize_text(title)

    if session is not None:
        return session.query(ComicDB).filter(
            ComicDB.titles.like(f"%{title}%")
        ).order_by(
            ComicDB.last_update.desc(), ComicDB.id
        ).all()

    with Session() as current_session:
        return current_session.query(ComicDB).filter(
            ComicDB.titles.like(f"%{title}%")
        ).order_by(
            ComicDB.last_update.desc(), ComicDB.id
        ).all()


def comics_by_identity_key(
    identity_key: str,
    session: SessionType = None,
) -> List[ComicDB]:
    if not identity_key:
        return []

    if session is not None:
        return session.query(ComicDB).filter(
            ComicDB.identity_key == identity_key
        ).order_by(ComicDB.id).all()

    with Session() as current_session:
        return current_session.query(ComicDB).filter(
            ComicDB.identity_key == identity_key
        ).order_by(ComicDB.id).all()


def canonical_comic_by_identity_key(
    identity_key: str,
    session: SessionType = None,
) -> ComicDB | None:
    matches = comics_by_identity_key(identity_key, session)
    if not matches:
        return None
    return matches[0]


def canonical_comic_by_title(
    title: str,
    com_type: int = 0,
    session: SessionType = None,
) -> ComicDB | None:
    identity_key = build_identity_key(title, com_type)
    return canonical_comic_by_identity_key(identity_key, session)


def canonical_comic_by_titles(
    titles: List[str] | str,
    com_type: int = 0,
    session: SessionType = None,
) -> ComicDB | None:
    identity_key = build_identity_key_from_titles(titles, com_type)
    return canonical_comic_by_identity_key(identity_key, session)


def comics_by_title_no_case(
    title: str, _from: int = 0, limit: int = 20,
    only_tracked: bool = False, only_unchecked: bool = False,
    full_query: bool = False
) -> (List[ComicDB], Pagination):
    title = normalize_text(title)
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
MERGE_SELF_ERROR = 'Cannot merge comic with itself'
MERGE_TYPE_ERROR = 'Comics to merge should be of the same type'
MERGE_CONFLICT_ERROR = (
    'Unable to merge comics because the merged record conflicts with an existing comic identity'
)
MERGE_DB_ERROR = 'Unable to merge comics due to a database error'


def _merge_title_variants(
    current_titles: Sequence[str] | str,
    incoming_titles: Sequence[str] | str,
    com_type: int,
) -> List[str]:
    return normalize_title_variants(
        [*list(current_titles), *list(incoming_titles)],
        com_type,
    )


def _merge_lookup_values(
    current_values: Iterable[int],
    incoming_values: Iterable[int],
) -> List[int]:
    return merge_unique_values(current_values, incoming_values)


def merge_comics(base_id: int, merging_id: int) -> tuple[dict | None, str | None, int]:
    '''>>> merge_comics(10, 24) -> ComicJSON: dict | None, error msg | None, status_code '''
    if base_id == merging_id:
        return None, MERGE_SELF_ERROR, 400
    with Session() as session:
        comic: ComicDB | None = session.query(ComicDB).get(base_id)
        if comic is None:
            return None, COMIC_NOT_FOUND.format(base_id), 404
        d_comic: ComicDB | None = session.query(ComicDB).get(merging_id)
        if d_comic is None:
            return None, COMIC_NOT_FOUND.format(merging_id), 404
        if d_comic.com_type != 0 and comic.com_type != d_comic.com_type:
            return None, MERGE_TYPE_ERROR, 400

        try:
            comic.set_titles(
                _merge_title_variants(
                    comic.get_titles(),
                    d_comic.get_titles(),
                    comic.com_type,
                )
            )
            comic.set_genres(
                _merge_lookup_values(
                    comic.get_genres(),
                    d_comic.get_genres(),
                )
            )
            comic.set_published_in(
                _merge_lookup_values(
                    comic.get_published_in(),
                    d_comic.get_published_in(),
                )
            )
            comic.current_chap = max(comic.current_chap, d_comic.current_chap)
            comic.viewed_chap = max(comic.viewed_chap, d_comic.viewed_chap)
            comic.track = int(bool(comic.track or d_comic.track))
            comic.rating = comic.rating or d_comic.rating
            comic.author = comic.author or d_comic.author
            comic.description = comic.description or d_comic.description
            comic.cover = comic.cover or d_comic.cover

            session.delete(d_comic)
            session.flush()
            comicJSON = comic.toJSON()
            session.commit()
        except IntegrityError as err:
            session.rollback()
            log.exception(
                'Merge conflict for comics %s <- %s: %s',
                base_id,
                merging_id,
                err,
            )
            return None, MERGE_CONFLICT_ERROR, 409
        except SQLAlchemyError as err:
            session.rollback()
            log.exception(
                'Database error merging comics %s <- %s: %s',
                base_id,
                merging_id,
                err,
            )
            return None, MERGE_DB_ERROR, 500

        rebuild_json_backup_from_db(persist_file=True)
        return comicJSON, None, 200
