# python tests/__main__.py

import unittest
import sys
sys.path.append("./src")
from scrap import com_type_parse
from db import Types, ComicDB
from db.helpers import manage_multi_finds

class TestScrapTypes(unittest.TestCase):
    def test_expected_conversion(self):
        self.assertEqual(com_type_parse('Manga'),Types['Manga'])
        self.assertEqual(com_type_parse('Manhua'),Types['Manhua'])
        self.assertEqual(com_type_parse('Manhwa'),Types['Manhwa'])
        self.assertEqual(com_type_parse('Novel'),Types['Novel'])
        self.assertEqual(com_type_parse('Comic'),Types['Manhwa'])
        self.assertEqual(com_type_parse('Random'),Types['Unknown'])
    def test_type_error(self):
        with self.assertRaises(AttributeError):
            com_type_parse(6.5)
    def test_ignoring_case(self):
        self.assertEqual(com_type_parse('NOVEL'),Types['Novel'])
        self.assertEqual(com_type_parse('manHua'),Types['Manhua'])

class TestScrapTitles(unittest.TestCase):
    def test_expected_conversion(self):
        novel = Types['Novel']
        manhwa = Types['Manhwa']
        title = 'Some novel/comic title'
        title_novel = title + ' - novel'
        db_comics = [
            ComicDB(None, title      , 0, com_type = manhwa),
            ComicDB(None, title_novel, 0, com_type = novel)
            ]
        db_comics, title = manage_multi_finds(db_comics, novel, title)
        self.assertEqual(len(db_comics),1)
        self.assertEqual(db_comics[0].titles,title_novel)
    def test_response_error(self):
        title = 'No error, testing more than 2 comics for 1 title'
        db_comics = [
            ComicDB(None, title      , 0),
            ComicDB(None, title + "1", 0),
            ComicDB(None, title + "2", 0)
            ]
        db_comics, title = manage_multi_finds(db_comics, 0, title)
        self.assertNotEqual(len(db_comics),1)
        self.assertEqual(db_comics[0].titles,title)
    def test_one_letter_diff(self):
        manhua = Types['Manhua']
        title = 'Soul land ii'
        title_expected = title + 'i'
        db_comics = [
            ComicDB(None, title         , 0, com_type = manhua),
            ComicDB(None, title_expected, 0, com_type = manhua)
            ]
        db_comics, title = manage_multi_finds(db_comics, manhua, title_expected)
        self.assertEqual(len(db_comics), 1)
        self.assertEqual(db_comics[0].titles, title_expected)

if __name__=='__main__':
	unittest.main()