## python db/repopulate_db.py

def manage_multi_finds(db_comics: list, com_type: int, title: str):
    ''' Takes a list of db_comics, if handles cases for 0, 1 and 2 items found.

    It returns an empty list to create a new comic or a 1 item list if the 
    comic match a register in the database '''
    if len(db_comics) > 2:
        print(f'Error, more than 2 comics found with title like: {title}')
    elif len(db_comics) == 2:
        # In case of two registers found, check if one is a novel
        if (db_comics[0].com_type == com_type and 
            db_comics[1].com_type != com_type):
            db_comics = [db_comics[0]]
        elif (db_comics[1].com_type == com_type and 
            db_comics[0].com_type != com_type):
            db_comics = [db_comics[1]]
        # If there is no novel, check inside the lists for exact match
        elif (title in db_comics[0].get_titles()):
            db_comics = [db_comics[0]]
        elif (title in db_comics[1].get_titles()):
            db_comics = [db_comics[1]]
        # If after previous check there is no exact match, it means
        # a new comic found
        else:
            db_comics = []
    elif len(db_comics) == 1 and (title not in db_comics[0].get_titles()):
            db_comics = []
    return db_comics