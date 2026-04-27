import unittest

from src.db import ComicDB, Types
from src.db.identity import (
    build_identity_key,
    normalize_title_variants,
    title_match_key,
    titles_are_prefix_match,
)
from src.scrape.scrapper import ScrapedComic, _normalize_comic_data


class TestIdentityHelpers(unittest.TestCase):
    def test_novel_and_comic_use_different_identity_keys(self):
        title = "A mercenary's rebirth among nobles"
        comic_key = build_identity_key(title, Types.Manhwa)
        novel_key = build_identity_key(title, Types.Novel)

        self.assertNotEqual(comic_key, novel_key)
        self.assertEqual(comic_key, "series:a mercenary's rebirth among nobles")
        self.assertEqual(novel_key, "novel:a mercenary's rebirth among nobles")

    def test_normalize_title_variants_enforces_novel_suffix_on_primary_title(self):
        titles = normalize_title_variants(
            ["A mercenary's rebirth among nobles (Novel)", "A mercenary's rebirth among nobles"],
            Types.Novel,
        )

        self.assertEqual(titles[0], "A mercenary's rebirth among nobles - novel")
        self.assertIn("A mercenary's rebirth among nobles", titles)

    def test_normalize_title_variants_uses_sentence_case_storage(self):
        titles = normalize_title_variants(
            ["THE WORLD'S BEST ENGINEER", "the world’s best engineer"],
            Types.Manhwa,
        )

        self.assertEqual(titles, ["The world's best engineer"])

    def test_title_match_key_ignores_leading_article_and_spacing(self):
        self.assertEqual(
            title_match_key("The holy emperor's grandson is a necromancer"),
            title_match_key("Holy emperor's grandsonis a necromancer"),
        )

    def test_titles_are_prefix_match_handles_truncated_source_title(self):
        self.assertTrue(
            titles_are_prefix_match(
                "The holy emperor's grandson is a necr",
                "Holy emperor's grandson is a necromancer",
            )
        )


class TestComicIdentity(unittest.TestCase):
    def test_comic_db_normalizes_titles_and_identity_key(self):
        comic = ComicDB(
            id=None,
            titles="The duke’s daughter tames the beast",
            current_chap=7,
            com_type=Types.Manhwa,
        )

        self.assertEqual(
            comic.get_titles()[0],
            "The duke's daughter tames the beast",
        )
        self.assertEqual(
            comic.identity_key,
            "series:the duke's daughter tames the beast",
        )

    def test_comic_db_recomputes_primary_title_when_type_changes(self):
        comic = ComicDB(
            id=None,
            titles="Mercenary rebirth",
            current_chap=12,
            com_type=Types.Manhwa,
        )

        comic.com_type = Types.Novel
        comic.normalize_titles()

        self.assertEqual(comic.get_titles()[0], "Mercenary rebirth - novel")
        self.assertEqual(comic.identity_key, "novel:mercenary rebirth")


class TestScrapeNormalization(unittest.TestCase):
    def test_scrape_normalization_uses_normalize_text(self):
        scraped = ScrapedComic(
            chapter="Chapter 21",
            title="A mercenary’s rebirth among nobles",
            cover_url="https://example.com/cover.webp",
            com_type="manhwa",
            status="ongoing",
        )

        normalized = _normalize_comic_data(scraped, 1)

        self.assertIsNotNone(normalized)
        self.assertEqual(
            normalized.get_titles()[0],
            "A mercenary's rebirth among nobles",
        )
        self.assertEqual(
            normalized.identity_key,
            "series:a mercenary's rebirth among nobles",
        )


if __name__ == "__main__":
    unittest.main()
