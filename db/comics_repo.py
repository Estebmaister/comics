from db.models import ComicDB, session

def all_comics(_from: int = 0, _limit: int = 20):
    return session.query(ComicDB).all()

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