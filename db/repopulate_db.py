# python db/repopulate_db.py
## Repopulates the DB with the json file as backup
from models import load_comics, ComicDB, save_comics_file, session
from helpers import manage_multi_finds

# Looping all comics
for comic in load_comics:
    first_title: str = comic['titles'][0]
    # Look for a certain title in DB
    db_comic = session.query(ComicDB).filter(
            ComicDB.titles.like(f"%{first_title}%")
        ).all()
    ## Check for multiple responses
    db_comic = manage_multi_finds(db_comic, comic['com_type'], first_title)
    
    if len(db_comic) == 0:
        new_db_comic = ComicDB(comic['id'], 
            '|'.join(comic['titles']), 
            comic['current_chap'],
            comic['cover'],
            comic['last_update'],
            comic['com_type'],
            comic['status'],
            '|'.join([str(p) for p in comic['published_in']]),
            '|'.join([str(g) for g in comic['genres']]),
            comic['description'],
            comic['author'],
            comic['track'],
            comic['viewed_chap']
            )
        session.add(new_db_comic)
        session.commit()
    elif len(db_comic) == 1:
        print(f'{comic["id"]} already in DB ({db_comic[0].id}) - '
                + f'{first_title} - {db_comic[0].get_titles()}')

save_comics_file(load_comics)