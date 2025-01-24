# python src/db/db_sqlite3.py
import os
import sqlite3

db_file = os.path.join(os.path.dirname(__file__), "comics.db")
conn = sqlite3.connect(db_file)
c = conn.cursor()
query = 'PRAGMA case_sensitive_like = true'
exec = c.execute(query)

like = "Boundless necromancer"
query = ''' SELECT * FROM comics WHERE titles LIKE '%{}%' '''. format(like)
query = ''' UPDATE comics SET titles = 'Relife player|Re:life player|Re: life player' WHERE id IN ('9') '''
query = ''' UPDATE comics SET id = 325 WHERE id = 330 '''
query = ''' DELETE FROM comics WHERE id > 680 '''
query = ''' UPDATE comics SET genres = "0" WHERE id = 83 '''
query = ''' SELECT * FROM comics WHERE titles LIKE '%{}%' '''. format(like)


exec = c.execute(query)
print(exec.fetchall())

conn.commit()
conn.close()
