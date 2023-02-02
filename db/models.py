import os, json, time
from enum import IntEnum, unique
from sqlalchemy import create_engine
from sqlalchemy.sql import text
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy import create_engine, Column, Integer, String

db_file = os.path.join(os.path.dirname(__file__), "comics.db")
engine = create_engine(
    f'sqlite:///{db_file}', 
    connect_args={'check_same_thread': False}
)

Base = declarative_base()

comic_file = os.path.join(os.path.dirname(__file__), "comics.json")
load_comics = []

with open(comic_file) as js_file:
    js_read_file = js_file.read()
    if js_read_file != "": load_comics = json.loads(js_read_file)
    load_comics.sort(key=lambda a : a["id"])

def save_comics_file(load_comics):
    with open(comic_file, "w") as js_file:
        js_file.write(json.dumps(load_comics, indent=2))

@unique
class Types(IntEnum):
    Unknown:int = 0
    Manga:  int = 1
    Manhua: int = 2
    Manhwa: int = 3
    Novel:  int = 4

@unique
class Statuses(IntEnum):
    Unknown:    int = 0
    Completed:  int = 1
    OnAir:      int = 2
    Break:      int = 3
    Dropped:    int = 4

@unique
class Genres(IntEnum):
    Unknown:        int = 0
    Action:         int = 1
    Adventure:      int = 2
    Fantasy:        int = 3
    Overpowered:    int = 4
    Comedy:         int = 5
    Drama:          int = 6
    SchoolLife:     int = 7
    System:         int = 8
    Supernatural:   int = 9
    MartialArts:    int = 10
    Romance:        int = 11
    Shounen:        int = 12
    Reincarnation:  int = 13

@unique
class Publishers(IntEnum):
    Unknown:        int = 0
    Asura:          int = 1
    ReaperScans:    int = 2
    ManhuaPlus:     int = 3
    FlameScans:     int = 4
    LuminousScans:  int = 5
    ResetScans:     int = 6
    IsekaiScan:     int = 7
    RealmScans:     int = 8

class ComicJSON(dict):
    def __init__(self,
        id:           int,
        titles:       list[str],
        current_chap: int,
        cover:        str = "",
        last_update:  int = int(time.time()),
        com_type:     Types = Types.Unknown,
        status:       Statuses = Statuses.Unknown,
        published_in: list[Publishers] = [Publishers.Unknown],
        genres:       list[Genres] = [Genres.Unknown],
        description:  str = "",
        author:       str = "",
        track:        bool = False,
        viewed_chap:  int = 0
        ):
        dict.__init__( self,
            id          =   id,
            titles      =   titles,
            current_chap=   current_chap,
            cover       =   cover,
            last_update =   last_update,
            com_type    =   com_type,
            status      =   status,
            published_in=   published_in,
            genres      =   genres,
            description =   description,
            author      =   author,
            track       =   track,
            viewed_chap =   viewed_chap
        )

class ComicDB(Base):
    __tablename__ = 'comics'
    id = Column(Integer, primary_key=True)
    titles = Column(String)
    current_chap = Column(Integer)
    cover = Column(String)
    last_update = Column(Integer)
    com_type = Column(Integer)
    status = Column(Integer)
    published_in = Column(String)
    genres = Column(String)
    description = Column(String)
    author = Column(String)
    track = Column(Integer)
    viewed_chap = Column(Integer)

    def __init__(self,
        id:           int,
        titles:       str,
        current_chap: int,
        cover:        str = "",
        last_update:  int = int(time.time()),
        com_type:     Types = Types.Unknown,
        status:       Statuses = Statuses.Unknown,
        published_in: str = "0",
        genres:       str = "0",
        description:  str = "",
        author:       str = "",
        track:        int = 0,
        viewed_chap:  int = 0
    ):
        self.id = id
        self.titles = titles
        self.current_chap = current_chap
        self.cover = cover
        self.last_update = last_update
        self.com_type = com_type
        self.status = status
        self.published_in = published_in
        self.genres = genres
        self.description = description
        self.author = author
        self.track = track
        self.viewed_chap = viewed_chap

    def get_titles(self):
        return self.titles.split("|")
    def set_titles(self, titles: list[str]):
        self.titles = "|".join(titles)

    def get_published_in(self):
        return [Publishers(int(p)) for p in self.published_in.split("|")]
    def set_published_in(self, pubs: list[Publishers]):
        self.published_in = "|".join([str(int(p)) for p in pubs])

    def get_genres(self):
        return [Genres(int(g)) for g in self.genres.split("|")]
    def set_genres(self, genres: list[Genres]):
        self.genres = "|".join([str(int(g)) for g in genres])

    def toJSON(self):
        return ComicJSON(
            self.id,
            self.get_titles(),
            self.current_chap,
            self.cover,
            self.last_update,
            Types(self.com_type),
            Statuses(self.status),
            self.get_published_in(),
            self.get_genres(),
            self.description,
            self.author,
            bool(self.track),
            self.viewed_chap
        )

Base.metadata.create_all(engine)
Session = sessionmaker(bind = engine)
session = Session()
session.execute(text('PRAGMA case_sensitive_like = true'))