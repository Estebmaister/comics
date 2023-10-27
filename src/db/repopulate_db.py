# python db/repopulate_db.py
## Repopulates the DB with the json file as backup
from __init__ import load_comics, ComicDB, Types, save_comics_file, session
from helpers import manage_multi_finds

counter = 1
counter_flag = False
ids_skipped = []
def counting_skips(comic_id: int):
    global counter
    global counter_flag
    if counter_flag and comic_id != counter:
        counter += 1
        counter_flag = False
    elif counter_flag:
        ids_skipped.pop()
        counter_flag = False

    if comic_id != counter:
        ids_skipped.append(counter)
        ids_skipped.append(counter+1)
        counter_flag = True
        counter += 2
    else:
        counter += 1

# Looping all comics
for comic in load_comics:
    counting_skips(comic['id'])
    first_title: str = comic['titles'][0]
    # Look for a certain title in DB
    db_comic = session.query(ComicDB).filter(
        ComicDB.titles.like(f"%{first_title}%")
    ).all()
    ## Check for multiple responses
    if len(db_comic) > 1:
        db_comic, _ = manage_multi_finds(
            db_comic, Types(comic['com_type']), first_title
        )
    
    if len(db_comic) == 0:
        new_db_comic = ComicDB(comic['id'], 
            '|'.join(comic['titles']), 
            comic['current_chap'],
            comic['cover'],
            comic['last_update'],
            comic['com_type'],
            comic['status'],
            comic['published_in'],
            comic['genres'],
            comic['description'],
            comic['author'],
            int(comic['track']),
            comic['viewed_chap']
            )
        session.add(new_db_comic)
        session.flush()
    elif len(db_comic) == 1:
        print(f'{comic["id"]} already in DB ({db_comic[0].id}) - '
                + f'{first_title} - {db_comic[0].get_titles()}')

print(f'{len(load_comics)} comics checked on the JSON back up')
print(f'IDs {ids_skipped} not found')
save_comics_file(load_comics)
session.commit()