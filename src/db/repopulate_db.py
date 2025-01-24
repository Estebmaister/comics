#!/usr/bin/env python3
"""
Database Repopulation Script

This script repopulates the database using a JSON file as backup. It handles:
- Sequential ID management and gap detection
- Duplicate comic detection
- Comic data insertion with proper field mapping

Usage:
    python3 repopulate_db.py

Returns:
    0 if successful, 1 if errors occurred
"""

import datetime
from typing import List, Optional

from sqlalchemy.exc import IntegrityError

# Import helper function for handling multiple comic finds
from src.db.helpers import manage_multi_finds

# Import database models and utilities
from . import ComicDB, Types, load_comics, save_comics_file, session


class IDTracker:
    """Manages comic ID sequence tracking and gap detection."""

    def __init__(self):
        self.counter = 1
        self.counter_flag = False
        self.ids_skipped = []

    def track_id(self, comic_id: int) -> None:
        """
        Tracks and manages gaps in comic ID sequence.

        Args:
            comic_id (int): The current comic's ID being processed

        This function maintains a sequential counter and identifies gaps in the ID sequence.
        When a gap is found, it's recorded in ids_skipped list.
        """
        if self.counter_flag and comic_id != self.counter:
            self.counter += 1
            self.counter_flag = False
        elif self.counter_flag:
            self.ids_skipped.pop()
            self.counter_flag = False

        if comic_id != self.counter:
            self.ids_skipped.append(self.counter)
            self.ids_skipped.append(self.counter + 1)
            self.counter_flag = True
            self.counter += 2
        else:
            self.counter += 1


def create_comic_db_instance(comic: dict) -> ComicDB:
    """
    Creates a ComicDB instance from a comic dictionary.

    Args:
        comic (dict): Dictionary containing comic data

    Returns:
        ComicDB: New database instance with comic data
    """
    dt_object = datetime.fromisoformat(comic['last_update'])
    return ComicDB(
        comic['id'],
        '|'.join(comic['titles']),
        comic['current_chap'],
        comic['cover'],
        int(dt_object.timestamp()),
        comic['com_type'],
        comic['status'],
        comic['published_in'],
        comic['genres'],
        comic['description'],
        comic['author'],
        int(comic['track']),
        comic['viewed_chap'],
        comic['rating'],
        comic['deleted']
    )


def find_existing_comic(title: str) -> List[ComicDB]:
    """
    Searches database for comics with matching title.

    Args:
        title (str): Comic title to search for

    Returns:
        List[ComicDB]: List of matching comics
    """
    return session.query(ComicDB).filter(
        ComicDB.titles.like(f"%{title}%")
    ).all()


def process_comic(comic: dict, session) -> bool:
    """
    Process a single comic entry.

    Args:
        comic (dict): Comic data to process
        session: SQLAlchemy session

    Returns:
        bool: True if comic was processed successfully, False otherwise
    """
    try:
        first_title: str = comic['titles'][0]

        # Search database for comics with similar titles
        db_comic = find_existing_comic(first_title)

        # Handle cases where multiple comics with similar titles are found
        if len(db_comic) > 1:
            db_comic, _ = manage_multi_finds(
                db_comic, Types(comic['com_type']), first_title
            )

        # Add new comic if not found in database
        if len(db_comic) == 0:
            new_db_comic = create_comic_db_instance(comic)
            session.add(new_db_comic)
            try:
                session.flush()
                print(f"Added new comic: {first_title} (ID: {comic['id']})")
                return True
            except IntegrityError as e:
                session.rollback()
                print(f"Comic ID {comic['id']} already exists in database")
                return False
        # Log if comic already exists in database
        elif len(db_comic) == 1:
            print(
                f"Comic already exists: {comic['id']} (DB ID: {db_comic[0].id}) - "
                f"{first_title} - {db_comic[0].get_titles()}"
            )
            return True

        return True

    except KeyError as e:
        print(f"Missing required field in comic data: {e}")
        return False
    except Exception as e:
        print(f"Error processing comic {comic.get('id', 'unknown')}: {e}")
        return False


def main() -> int:
    """
    Main function to handle database repopulation.

    Returns:
        int: 0 if successful, 1 if errors occurred
    """
    try:
        id_tracker = IDTracker()
        comics_processed = 0
        comics_added = 0
        comics_existing = 0
        comics_failed = 0

        # Process each comic from the JSON backup
        for comic in load_comics:
            # Check for gaps in ID sequence
            id_tracker.track_id(comic['id'])

            # Process the comic
            if process_comic(comic, session):
                comics_processed += 1
                if len(find_existing_comic(comic['titles'][0])) == 0:
                    comics_added += 1
                else:
                    comics_existing += 1
            else:
                comics_failed += 1

        # Print summary statistics
        print(f"Comics processed: {comics_processed}")
        print(f"Comics added: {comics_added}")
        print(f"Comics already existing: {comics_existing}")
        print(f"Comics failed: {comics_failed}")
        print(f"IDs not found: {id_tracker.ids_skipped}")

        # Save updated comics to JSON file
        save_comics_file(load_comics)

        # Commit all changes to database
        session.commit()
        print("Database update completed successfully")
        return 0

    except Exception as e:
        print(f"Fatal error: {e}")
        session.rollback()
        return 1


if __name__ == "__main__":
    exit(main())
