from db import ComicDB, Types, session, load_comics, save_comics_file

def all_comics(_from: int = 0, _limit: int = 20, full: bool = False):
    if full:
        return session.query(ComicDB).all()
    else:
        return session.query(ComicDB).order_by("id").offset(_from).limit(_limit)

def comic_by_id(id: int):
    return session.query(ComicDB).get(id), session

def comics_by_title(title: str):
    return session.query(ComicDB).filter(
            ComicDB.titles.like(f"%{title}%")
        ).all(), session

def comics_by_title_no_case(title: str):
    return session.query(ComicDB).filter(
            ComicDB.titles.ilike(f"%{title.lower()}%")
        ).all()

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