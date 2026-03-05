from sqlalchemy.sql import text

from db import DB_ENGINE, Engines, Session


SQLITE_INDEXES = [
    "CREATE INDEX IF NOT EXISTS idx_comics_titles ON comics(titles)",
    "CREATE INDEX IF NOT EXISTS idx_comics_author ON comics(author)",
    "CREATE INDEX IF NOT EXISTS idx_comics_last_update ON comics(last_update DESC)",
    "CREATE INDEX IF NOT EXISTS idx_comics_track_status ON comics(track, status)",
    "CREATE INDEX IF NOT EXISTS idx_comics_tracked_updated ON comics(track, last_update DESC)",
]

POSTGRES_INDEXES = [
    "CREATE INDEX IF NOT EXISTS idx_comics_titles ON comics(titles)",
    "CREATE INDEX IF NOT EXISTS idx_comics_author ON comics(author)",
    "CREATE INDEX IF NOT EXISTS idx_comics_last_update ON comics(last_update DESC)",
    "CREATE INDEX IF NOT EXISTS idx_comics_track_status ON comics(track, status)",
    "CREATE INDEX IF NOT EXISTS idx_comics_tracked_updated ON comics(track, last_update DESC)",
]


def _indexes_for_engine():
    if DB_ENGINE == Engines.POSTGRES:
        return POSTGRES_INDEXES
    return SQLITE_INDEXES


def run() -> int:
    statements = _indexes_for_engine()
    with Session() as session:
        for stmt in statements:
            session.execute(text(stmt))
        session.commit()
    return len(statements)


if __name__ == "__main__":
    count = run()
    print(f"Applied {count} indexes.")
