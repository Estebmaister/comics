import json
import os
import signal
import sys
import time
from datetime import datetime, timezone
from enum import IntEnum, unique
from typing import Any, Dict, List, Union

from dotenv import load_dotenv
from flask_restx import Model
from flask_restx import fields as sf
from sqlalchemy import (URL, Boolean, Column, Integer, Sequence, String,
                        create_engine)
from sqlalchemy.orm import declarative_base, sessionmaker
from sqlalchemy.sql import text


@unique
class Engines(IntEnum):
    SQLITE = 1
    POSTGRES = 2
    MYSQL = 3
    MONGO = 4


# Engines to DB_ENGINE switch
engines = {
    '':             Engines.SQLITE,
    'sqlite':       Engines.SQLITE,
    'postgresql':   Engines.POSTGRES,
    'postgres':     Engines.POSTGRES,
    'mysql':        Engines.MYSQL,
    'mongo':        Engines.MONGO
}

load_dotenv()
try:
    DB_ENGINE: Engines = engines[os.getenv('DB_ENGINE')]
except:
    print('DB_ENGINE env var malformed, defaulting to sqlite')
    DB_ENGINE: Engines = Engines.SQLITE
DB_USER: str = os.getenv('DB_USER', 'esteb')
DB_NAME: str = os.getenv('DB_NAME', 'comics')
DB_PASS: str = os.getenv('DB_PASS', 'My0th3rS3lf')
DB_HOST: str = os.getenv('DB_HOST', '127.0.0.1')
DB_PORT: str = os.getenv('DB_PORT', '3306')
db_file = os.path.join(os.path.dirname(__file__), "comics.db")
ssl_ca = os.path.join(os.path.dirname(__file__),
                      "DigiCertGlobalRootG2.crt.pem")
comic_file = os.path.join(os.path.dirname(__file__), "comics.json")
load_comics = []

with open(comic_file) as js_file:
    js_read_file = js_file.read()
    if js_read_file != "":
        load_comics = json.loads(js_read_file)
    load_comics.sort(key=lambda a: a["id"])


def save_comics_file(load_comics):
    with open(comic_file, "w") as js_file:
        js_file.write(json.dumps(load_comics, indent=2))


def _engine_creation():
    DB_URL: Union[URL, str]
    CONNECT_ARGS: Dict[Any, Any]

    if DB_ENGINE == Engines.POSTGRES:
        DB_URL = URL.create(
            drivername='postgresql',
            username=DB_USER,
            password=DB_PASS,
            host=DB_HOST,
            port=int(DB_PORT),
            database=DB_NAME
        )
        CONNECT_ARGS = {}

    elif DB_ENGINE == Engines.MYSQL:
        DB_URL = URL.create(
            drivername='mysql+pymysql',
            username=DB_USER,
            password=DB_PASS,
            host=DB_HOST,
            port=int(DB_PORT),
            database=DB_NAME
        )
        CONNECT_ARGS = {'ssl': {'ca': ssl_ca, 'check_hostname': True}}

    else:
        DB_URL = f'sqlite:///{db_file}'
        CONNECT_ARGS = {'check_same_thread': False}

    return create_engine(DB_URL, connect_args=CONNECT_ARGS)


engine = _engine_creation()
print(engine)


seq = Sequence('comic_id_seq')
Base = declarative_base()
Session = sessionmaker(bind=engine)
session = Session()
if not DB_ENGINE or DB_ENGINE == Engines.SQLITE:
    session.execute(text('PRAGMA case_sensitive_like = true'))


@unique
class Types(IntEnum):
    Unknown = 0
    Manga = 1
    Manhua = 2
    Manhwa = 3
    Novel = 4


@unique
class Statuses(IntEnum):
    Unknown = 0
    Completed = 1
    OnAir = 2
    Break = 3
    Dropped = 4


@unique
class Genres(IntEnum):
    Unknown = 0
    Action = 1
    Adventure = 2
    Fantasy = 3
    Overpowered = 4
    Comedy = 5
    Drama = 6
    SchoolLife = 7
    System = 8
    Supernatural = 9
    MartialArts = 10
    Romance = 11
    Shounen = 12
    Reincarnation = 13


@unique
class Publishers(IntEnum):
    Unknown = 0
    Asura = 1
    ReaperScans = 2
    ManhuaPlus = 3
    FlameScans = 4
    LuminousScans = 5
    ResetScans = 6
    IsekaiScan = 7
    RealmScans = 8
    LeviatanScans = 9
    NightScans = 10
    VoidScans = 11
    DrakeScans = 12
    NovelMic = 13
    Mangagreat = 14
    Mangageko = 15
    Mangarolls = 16
    Manganato = 17
    FirstKiss = 18


# SQLAlchemy model definition
class ComicDB(Base):
    """
    Comic database model representing a comic entry.

    Attributes:
        id (int): Unique identifier for the comic
        titles (str): Pipe-separated list of titles
        current_chap (int): Current chapter number
        ...
    """
    __tablename__ = 'comics'
    id = Column(
        Integer, primary_key=True,
        autoincrement='auto' if Engines.MYSQL == DB_ENGINE else False,
        server_default=None if Engines.MYSQL == DB_ENGINE else seq.next_value()
    )
    titles = Column(String(2083),   nullable=False)
    description = Column(String(2000), default="")
    author = Column(String(150),    default="")
    cover = Column(String(2083),    default="")
    last_update = Column(Integer,   default=lambda: int(time.time()))
    published_in = Column(String(50), default="0")
    genres = Column(String(50),     default="0")
    com_type = Column(Integer,      nullable=False, default=0)
    status = Column(Integer,        nullable=False, default=0)
    current_chap = Column(Integer,  nullable=False, default=0)
    viewed_chap = Column(Integer,   nullable=False, default=0)
    track = Column(Integer,         nullable=False, default=0)
    rating = Column(Integer,        nullable=False, default=0)
    deleted = Column(Boolean,       nullable=False, default=False)

    def __init__(
        self,
        id:           Column[int],  # required
        titles:       str,  # required
        current_chap: int,  # required
        cover:        str = "",
        last_update:  int = int(time.time()),
        com_type:     int = 0,
        status:       int = 0,
        published_in: str = "0",
        genres:       str = "0",
        description:  str = "",
        author:       str = "",
        track:        int = 0,
        viewed_chap:  int = 0,

    ):
        self.id = id
        self.titles = str(titles)
        self.current_chap = int(current_chap)
        self.cover = str(cover)
        self.last_update = int(last_update)
        self.com_type = int(com_type)
        self.status = int(status)
        if isinstance(published_in, list):
            self.set_published_in(published_in)
        else:
            self.published_in = str(int(published_in))
        if isinstance(genres, list):
            self.set_genres(genres)
        else:
            self.genres = str(int(genres))
        self.description = str(description)
        self.author = str(author)
        self.track = int(track)
        self.viewed_chap = int(viewed_chap)
        self.rating = 0
        self.deleted = False

    def get_titles(self) -> List[str]:
        return str(self.titles).split("|")

    def set_titles(self, titles: Union[str, List[str]]) -> None:
        if type(titles) is list:
            self.titles = str("|".join(titles))
        elif type(titles) is str:
            self.titles = str(titles)

    def get_published_in(self) -> List[Publishers]:
        return [Publishers(int(p)) for p in str(self.published_in).split("|")]

    def set_published_in(self, pubs: List[Publishers]) -> None:
        self.published_in = "|".join([str(int(p)) for p in pubs])

    def get_genres(self) -> List[Genres]:
        return [Genres(int(g)) for g in str(self.genres).split("|")]

    def set_genres(self, genres: List[Genres]) -> None:
        self.genres = "|".join([str(int(g)) for g in genres])

    def toJSON(self) -> dict:
        last_update: str = datetime.fromtimestamp(
            self.last_update, tz=timezone.utc).isoformat()
        return dict(
            id=self.id,
            titles=self.get_titles(),
            current_chap=self.current_chap,
            cover=self.cover,
            last_update=last_update,
            com_type=Types(self.com_type),
            status=Statuses(self.status),
            published_in=self.get_published_in(),
            genres=self.get_genres(),
            description=self.description,
            author=self.author,
            track=bool(self.track),
            viewed_chap=self.viewed_chap,
            rating=self.rating,
            deleted=self.deleted
        )


comic_swagger_model = Model('Comic', {
    'id':           sf.Integer(readonly=True, description='Comic unique identifier'),
    'titles':       sf.List(sf.String(), required=True, description='Comic titles'),
    'current_chap': sf.Integer(required=True, description='Comic current chapter'),
    'cover':        sf.String(required=True, description='Comic cover img'),
    'published_in': sf.List(sf.Integer(), description='Comic publishers, ex: [1]'),
    'author':       sf.String(description='Comic author', default=''),
    'description':  sf.String(description='Comic details', default=''),
    'com_type':     sf.Integer(description='Comic type, ex: 3 =Manhwa'),
    'status':       sf.Integer(description='Comic status, ex: 2 =OnAir'),
    'last_update':  sf.Date(description='Comic last update, ex: int(time.now())'),
    'genres':       sf.List(sf.Integer(), description='Comic genres, ex: [6] =Drama'),
    'track':        sf.Boolean(description='Comic track'),
    'viewed_chap':  sf.Integer(description='Comic viewed chapter')
})

# Create tables if they don't exist
Base.metadata.create_all(engine)

if DB_ENGINE == Engines.POSTGRES:
    seq.create(bind=engine)
    last_record = session.query(ComicDB).order_by(ComicDB.id.desc()).first()
    last_id = last_record.id if last_record else 0
    session.execute(text(f"SELECT setval('comic_id_seq', {last_id})"))

db_classes_file = os.path.join(os.path.dirname(__file__), "db_classes.json")


def save_db_classes_file():
    with open(db_classes_file, "w") as js_file:
        classes: dict[str, list[str]] = {}
        classes.update({'com_type': [data.name for data in Types]})
        classes.update({'status': [data.name for data in Statuses]})
        classes.update({'genres': [data.name for data in Genres]})
        classes.update({'published_in': [data.name for data in Publishers]})
        js_file.write(json.dumps(classes, indent=2))


save_db_classes_file()


def close_signal_handler(sig, frame):
    session.close()
    print('\nDB connection closed...')
    sys.exit(0)


signal.signal(signal.SIGINT, close_signal_handler)
