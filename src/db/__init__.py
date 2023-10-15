import os, json, time, signal, sys
from enum import IntEnum, unique
from sqlalchemy import create_engine
from sqlalchemy.sql import text
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy import create_engine, engine, Column, Integer, String
from flask_restx import fields as sf
from dotenv import load_dotenv

load_dotenv()

db_file = os.path.join(os.path.dirname(__file__), "comics.db")
engine = create_engine( f'sqlite:///{db_file}', 
    connect_args={'check_same_thread': False} )

DB_DRIVER: str = 'postgresql' # 'sqlite'
DB_USER: str = 'esteb'
DB_PASS: str = os.getenv('PGPASSWORD', 'My0therSelf')
DB_HOST: str = os.getenv('PGHOST', '127.0.0.1')
DB_PORT: int = os.getenv('PGPORT', '5432')
DB_NAME: str = 'comics'
DB_URL: str  = engine.url.create( drivername=DB_DRIVER, username=DB_USER,
    password=DB_PASS, host=DB_HOST, port=DB_PORT, database=DB_NAME )
print(DB_URL)

if os.getenv('PRODUCTION', False): engine = create_engine(DB_URL)

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
    LeviatanScans:  int = 9
    NightScans:     int = 10
    VoidScans:      int = 11

class ComicDB(Base):
    __tablename__ = 'comics'
    id            = Column(Integer, primary_key=True)
    titles        = Column(String)
    current_chap  = Column(Integer)
    cover         = Column(String)
    last_update   = Column(Integer)
    com_type      = Column(Integer)
    status        = Column(Integer)
    published_in  = Column(String)
    genres        = Column(String)
    description   = Column(String)
    author        = Column(String)
    track         = Column(Integer)
    viewed_chap   = Column(Integer)

    def __init__(self,
        id:           int, #required
        titles:       str, #required
        current_chap: int, #required
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
        self.id           = id
        self.titles       = titles
        self.current_chap = current_chap
        self.cover        = cover
        self.last_update  = last_update
        self.com_type     = com_type
        self.status       = status
        self.published_in = published_in
        self.genres       = genres
        self.description  = description
        self.author       = author
        self.track        = track
        self.viewed_chap  = viewed_chap

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
        return dict(
            id           = self.id,
            titles       = self.get_titles(),
            current_chap = self.current_chap,
            cover        = self.cover,
            last_update  = self.last_update,
            com_type     = Types(self.com_type),
            status       = Statuses(self.status),
            published_in = self.get_published_in(),
            genres       = self.get_genres(),
            description  = self.description,
            author       = self.author,
            track        = bool(self.track),
            viewed_chap  = self.viewed_chap
        )

swagger_model = {
'id':          sf.Integer(readonly=True, description='Comic unique identifier'),
'titles':      sf.List(sf.String(),required=True, description='Comic titles'),
'current_chap':sf.Integer(required=True, description='Comic current chapter'),
'cover':       sf.String(required=True, description='Comic cover img'),
'published_in':sf.List(sf.Integer(),description='Comic publishers, ex: [1]'),
'author':      sf.String(description='Comic author' ,default=''),
'description': sf.String(description='Comic details',default=''),
'com_type':    sf.Integer(description='Comic type, ex: 3 =Manhwa'),
'status':      sf.Integer(description='Comic status, ex: 2 =OnAir'),
'last_update': sf.Integer(description='Comic last update, ex: int(time.now())'),
'genres':      sf.List(sf.Integer(),description='Comic genres, ex: [6] =Drama'),
'track':       sf.Boolean(description='Comic track'),
'viewed_chap': sf.Integer(description='Comic viewed chapter')
}

Base.metadata.create_all(engine)
Session = sessionmaker(bind = engine)
session = Session()
session.execute(text('PRAGMA case_sensitive_like = true'))

def close_signal_handler(sig, frame):
    session.close()
    print('\nDB connection closed...')
    sys.exit(0)

signal.signal(signal.SIGINT, close_signal_handler)