import os
import sqlite3
import tempfile
import unittest
from unittest.mock import patch

from db import _comic_db_from_json_record, _rebuild_sqlite_from_json, _sqlite_integrity_ok
from src.db.scraped_register import _parse_chapter_number


class TestChapterParsing(unittest.TestCase):
    def test_parse_plain_integer(self):
        self.assertEqual(_parse_chapter_number('147'), 147)

    def test_parse_chapter_label(self):
        self.assertEqual(_parse_chapter_number('Chapter 21'), 21)

    def test_parse_season_and_chapter(self):
        self.assertEqual(_parse_chapter_number('Season 2 Chapter 45'), 45)

    def test_parse_missing_digits_returns_none(self):
        self.assertIsNone(_parse_chapter_number('Prologue'))


class TestSqliteRecovery(unittest.TestCase):
    def test_integrity_check_detects_corrupt_file(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
            db_path = os.path.join(tmp_dir, 'comics.db')
            with open(db_path, 'wb') as db_file:
                db_file.write(b'not a sqlite database')

            with patch('db.db_file', db_path):
                self.assertFalse(_sqlite_integrity_ok())

    def test_rebuild_restores_records_from_json_backup(self):
        comic = {
            'id': 1,
            'titles': 'Example comic',
            'current_chap': 12,
            'cover': '',
            'last_update': 1_700_000_000,
            'com_type': 3,
            'status': 2,
            'published_in': '1',
            'genres': '0',
            'description': '',
            'author': '',
            'track': 0,
            'viewed_chap': 0,
            'rating': 0,
            'deleted': False,
            'cover_visible': True,
        }

        with tempfile.TemporaryDirectory() as tmp_dir:
            db_path = os.path.join(tmp_dir, 'comics.db')
            with open(db_path, 'wb') as db_file:
                db_file.write(b'not a sqlite database')

            with patch('db.db_file', db_path), patch('db.load_comics', [comic]):
                _rebuild_sqlite_from_json()

            with sqlite3.connect(db_path) as connection:
                integrity = connection.execute('PRAGMA integrity_check').fetchone()[0]
                row = connection.execute(
                    'SELECT titles, current_chap, identity_key FROM comics'
                ).fetchone()

            self.assertEqual(integrity, 'ok')
            self.assertEqual(row[0], 'Example comic')
            self.assertEqual(row[1], 12)
            self.assertEqual(row[2], 'series:example comic')

    def test_json_record_helper_sets_identity_key(self):
        comic = _comic_db_from_json_record(
            {
                'id': 9,
                'titles': 'Example comic',
                'current_chap': 3,
                'cover': '',
                'last_update': 1_700_000_000,
                'com_type': 3,
                'status': 2,
                'published_in': '1',
                'genres': '0',
            }
        )
        self.assertEqual(comic.identity_key, 'series:example comic')


if __name__ == '__main__':
    unittest.main()
