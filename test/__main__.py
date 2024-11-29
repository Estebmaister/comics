# python tests/__main__.py

from scrape import _parse_type
from db.helpers import manage_multi_finds
from db import ComicDB, Types
import sys
import unittest

sys.path.append("./src")


class TestScrapTypes(unittest.TestCase):
    def test_expected_conversion(self):
        self.assertEqual(_parse_type('Manga'), Types['Manga'])
        self.assertEqual(_parse_type('Manhua'), Types['Manhua'])
        self.assertEqual(_parse_type('Manhwa'), Types['Manhwa'])
        self.assertEqual(_parse_type('Novel'), Types['Novel'])
        self.assertEqual(_parse_type('Comic'), Types['Manhwa'])
        self.assertEqual(_parse_type('Random'), Types['Unknown'])

    def test_type_error(self):
        with self.assertRaises(AttributeError):
            _parse_type(6.5)

    def test_ignoring_case(self):
        self.assertEqual(_parse_type('NOVEL'), Types['Novel'])
        self.assertEqual(_parse_type('manHua'), Types['Manhua'])


class TestScrapTitles(unittest.TestCase):
    def test_expected_conversion(self):
        novel = Types['Novel']
        manhwa = Types['Manhwa']
        title = 'Some novel/comic title'
        title_novel = title + ' - novel'
        db_comics = [
            ComicDB(None, title, 0, com_type=manhwa),
            ComicDB(None, title_novel, 0, com_type=novel)
        ]
        db_comics, title = manage_multi_finds(db_comics, novel, title)
        self.assertEqual(len(db_comics), 1)
        self.assertEqual(db_comics[0].titles, title_novel)

    def test_response_error(self):
        title = '"No Error", testing alert for more than 2 comics with 1 title'
        db_comics = [
            ComicDB(None, title, 0),
            ComicDB(None, title + "1", 0),
            ComicDB(None, title + "2", 0)
        ]
        db_comics, title = manage_multi_finds(db_comics, 0, title)
        self.assertNotEqual(len(db_comics), 1)
        self.assertEqual(db_comics[0].titles, title)

    def test_one_letter_diff(self):
        manhua = Types['Manhua']
        title = 'Soul land ii'
        title_expected = title + 'i'
        db_comics = [
            ComicDB(None, title, 0, com_type=manhua),
            ComicDB(None, title_expected, 0, com_type=manhua)
        ]
        db_comics, title = manage_multi_finds(
            db_comics, manhua, title_expected)
        self.assertEqual(len(db_comics), 1)
        self.assertEqual(db_comics[0].titles, title_expected)


if __name__ == '__main__':
    unittest.main()
