import math
from db import ComicDB, Types, session, load_comics, save_comics_file 
# TODO: pagination object

def all_comics(_from: int = 0, limit: int = 20, 
        only_tracked: bool = False, only_unchecked: bool = False,
        full_query: bool = False
    ):
    if full_query:
        return session.query(ComicDB).all()
    
    partial_result = session.query(ComicDB).order_by(
        ComicDB.last_update.desc(), ComicDB.id
    )
    if only_tracked:
        if only_unchecked:
            partial_result = partial_result.filter(
                ComicDB.track == True,
                ComicDB.current_chap != ComicDB.viewed_chap
            )
        else:
            partial_result = partial_result.filter(ComicDB.track == True)
    total = partial_result.count()
    total_pages = math.ceil(total/limit)
    current_page = math.ceil(_from/limit +1)
    pagination_data = {
        'from': _from, 'limit': limit,
        'total': total, 'total_pages': total_pages, 
        'current_page': current_page
    }
    return partial_result.offset(_from).limit(limit), pagination_data

def comic_by_id(id: int):
    return session.query(ComicDB).get(id), session

def comics_by_title(title: str):
    return session.query(ComicDB).filter(
            ComicDB.titles.like(f"%{title}%")
        ).order_by(
            ComicDB.last_update.desc(), ComicDB.id
        ).all(), session

def comics_by_title_no_case(
        title: str, _from: int = 0, limit: int = 20,
        only_tracked: bool = False, only_unchecked: bool = False,
        full_query: bool = False
    ):
    partial_result = session.query(ComicDB).filter(
            ComicDB.titles.ilike(f"%{title.lower()}%")
        ).order_by(
            ComicDB.last_update.desc(), ComicDB.id
        )
    if only_tracked:
        if only_unchecked:
            partial_result = partial_result.filter(
                ComicDB.track == True,
                ComicDB.current_chap != ComicDB.viewed_chap
            )
        else:
            partial_result = partial_result.filter(ComicDB.track == True)
    if full_query:
        return partial_result.all()

    total = partial_result.count()
    total_pages = math.ceil(total/limit)
    current_page = math.ceil(_from/limit +1)
    pagination_data = {
        'from': _from, 'limit': limit,
        'total': total, 'total_pages': total_pages, 
        'current_page': current_page
    }
    return partial_result.offset(_from).limit(limit), pagination_data

COMIC_NOT_FOUND = 'Comic {} not found'
def merge_comics(base_id: int, merging_id: int):
    '''>>> merge_comics(10, 24) -> Comic.toJSON(), None '''
    comic, session = comic_by_id(base_id)
    if comic is None: return None, COMIC_NOT_FOUND.format(base_id)
    d_comic = session.query(ComicDB).get(merging_id)
    if d_comic is None: return None, COMIC_NOT_FOUND.format(merging_id)
    if d_comic.com_type != 0 and comic.com_type != d_comic.com_type:
        return None, 'Comics to merge should be of the same type'
    json_comic = [com for com in load_comics if comic.id == com["id"]][0]
    dj_comic = [com for com in load_comics if d_comic.id == com["id"]][0]

    titles = list(set(comic.get_titles() + d_comic.get_titles()))
    comic.set_titles(titles)
    genres = list(set(comic.get_genres() + d_comic.get_genres()))
    comic.set_genres(genres)
    publishers = list(set(comic.get_published_in() +d_comic.get_published_in()))
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