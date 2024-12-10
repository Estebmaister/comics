#!/usr/bin/env python3

import json
import os
import sys
from datetime import datetime

from sqlalchemy.orm import Session

from . import ComicDB, engine

# Add project root to Python path
project_root = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "../"))
sys.path.insert(0, project_root)


def backup_database():
    try:
        # Create backups directory if it doesn't exist
        backup_dir = os.path.join(os.path.dirname(__file__), "backups")
        os.makedirs(backup_dir, exist_ok=True)

        # Generate backup filename with timestamp
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        backup_file = os.path.join(
            backup_dir, f"comics_backup_{timestamp}.json")

        # Retrieve all comics from database
        with Session(engine) as session:
            comics = session.query(ComicDB).all()
            comics_data = [comic.toJSON() for comic in comics]

        # Save to JSON file
        with open(backup_file, "w", encoding="utf-8") as f:
            json.dump(comics_data, indent=2, ensure_ascii=False, fp=f)

        print(f"Database backup created successfully: {backup_file}")
        return backup_file
    except Exception as e:
        print(f"Error creating backup: {str(e)}")
        raise


if __name__ == "__main__":
    backup_database()
