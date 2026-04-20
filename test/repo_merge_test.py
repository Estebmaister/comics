import unittest

from src.db import Types
from src.db.repo import _merge_title_variants


class TestMergeRepoHelpers(unittest.TestCase):
    def test_merge_title_variants_preserves_base_title_order(self):
        merged = _merge_title_variants(
            ["Base title", "Alt one"],
            ["Alt one", "Alt two"],
            Types.Manhwa,
        )

        self.assertEqual(merged, ["Base title", "Alt one", "Alt two"])


if __name__ == "__main__":
    unittest.main()
