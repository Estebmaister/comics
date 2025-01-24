#!/usr/bin/env python3
"""
Database Repopulation Script

This script updates the database and JSON file backup.

Usage:
    python3 db_update.py

Returns:
    0 if successful, 1 if errors occurred
"""

from datetime import datetime

from sqlalchemy.sql import text

from . import load_comics, save_comics_file, session

db_name = "comics"

# Old and new column names
old_column_name = "old_column"
new_column_name = "new_column"


def add_columns_to_db() -> None:
    # Add rating column
    query = '''ALTER TABLE {} ADD COLUMN rating INTEGER DEFAULT 0;
    '''.format(db_name)
    session.execute(text(query))

    # Add deleted column
    query = '''ALTER TABLE {} ADD COLUMN deleted BOOLEAN DEFAULT false;
    '''.format(db_name)
    session.execute(text(query))

    # Commit all changes to database
    session.commit()
    print("Database update completed successfully")


def add_columns_to_json() -> None:
    # Process each comic from the JSON backup
    for comic in load_comics:
        comic['last_update'] = datetime.fromtimestamp(
            comic['last_update']).isoformat()
        comic['rating'] = 0
        comic['deleted'] = False

    # Save updated comics to JSON file
    save_comics_file(load_comics)
    print("JSON update completed successfully")


def main() -> int:
    """
    Main function to handle database update.

    Returns:
        int: 0 if successful, 1 if errors occurred
    """
    try:
        add_columns_to_db()
        add_columns_to_json()

        # Query the schema of the 'comics' table
        execution = session.execute(text("PRAGMA table_info(comics);"))
        columns = execution.fetchall()
        print("Columns in the 'comics' table:")
        for column in columns:
            print(column)

        return 0

    except Exception as e:
        print(f"Fatal error: {e}")
        session.rollback()
        return 1


if __name__ == "__main__":
    exit(main())
