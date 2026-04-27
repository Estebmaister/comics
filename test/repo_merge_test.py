import unittest

from src.db import ComicDB, Types
from src.db.repo import _merge_cover_values, _merge_title_variants


class TestMergeRepoHelpers(unittest.TestCase):
    def test_merge_title_variants_preserves_base_title_order(self):
        merged = _merge_title_variants(
            ["Base title", "Alt one"],
            ["Alt one", "Alt two"],
            Types.Manhwa,
        )

        self.assertEqual(merged, ["Base title", "Alt one", "Alt two"])

    def test_merge_cover_values_prefers_visible_duplicate_cover(self):
        base = ComicDB(None, "Base", 1, cover="https://example.com/base.webp")
        base.cover_visible = False
        duplicate = ComicDB(None, "Duplicate", 1, cover="https://example.com/dup.webp")
        duplicate.cover_visible = True

        cover, cover_visible = _merge_cover_values(base, duplicate)

        self.assertEqual(cover, "https://example.com/dup.webp")
        self.assertEqual(cover_visible, True)

    def test_merge_cover_values_keeps_visible_base_cover(self):
        base = ComicDB(None, "Base", 1, cover="https://example.com/base.webp")
        base.cover_visible = True
        duplicate = ComicDB(None, "Duplicate", 1, cover="https://example.com/dup.webp")
        duplicate.cover_visible = True

        cover, cover_visible = _merge_cover_values(base, duplicate)

        self.assertEqual(cover, "https://example.com/base.webp")
        self.assertEqual(cover_visible, True)


if __name__ == "__main__":
    unittest.main()
