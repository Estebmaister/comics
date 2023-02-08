# python tests/__main__.py

import unittest
import sys
sys.path.append("./src")
from scrap import com_type_parse
from db import Types

class TestPrime(unittest.TestCase):
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

if __name__=='__main__':
	unittest.main()