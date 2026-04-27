import asyncio
import unittest

from src.db import ComicDB, Publishers, Session, Types
from src.db.helpers import manage_multi_finds
from src.db.repo import comics_by_title_prefix
from src.scrape.scrapper import ScrapedComic, _parse_type, _should_update_cover, register_comic


class TestScrapTypes(unittest.TestCase):
    def test_expected_conversion(self):
        self.assertEqual(_parse_type('Manga'), Types.Manga)
        self.assertEqual(_parse_type('Manhua'), Types.Manhua)
        self.assertEqual(_parse_type('Manhwa'), Types.Manhwa)
        self.assertEqual(_parse_type('Novel'), Types.Novel)
        self.assertEqual(_parse_type('Comic'), Types.Unknown)
        self.assertEqual(_parse_type('Random'), Types.Unknown)

    def test_type_error(self):
        with self.assertRaises(AttributeError):
            _parse_type(6.5)

    def test_ignoring_case(self):
        self.assertEqual(_parse_type('NOVEL'), Types.Novel)
        self.assertEqual(_parse_type('manHua'), Types.Manhua)


class TestScrapTitles(unittest.TestCase):
    def setUp(self):
        with Session() as session:
            session.query(ComicDB).delete()
            session.commit()

    def test_expected_conversion(self):
        novel = Types.Novel
        manhwa = Types.Manhwa
        title = 'Some novel/comic title'
        title_novel = title + ' - novel'
        db_comics = [
            ComicDB(None, title, 0, com_type=manhwa),
            ComicDB(None, title_novel, 0, com_type=novel)
        ]
        db_comics, title = manage_multi_finds(db_comics, novel, title)
        self.assertEqual(len(db_comics), 1)
        self.assertEqual(db_comics[0].titles, title)

    def test_response_error(self):
        title = '"No Error", testing alert for more than 2 comics with 1 title'
        db_comics = [
            ComicDB(None, title, 0),
            ComicDB(None, title + "1", 0),
            ComicDB(None, title + "2", 0)
        ]
        db_comics, title = manage_multi_finds(db_comics, 0, title)
        self.assertNotEqual(len(db_comics), 1)
        self.assertEqual(db_comics, [])
        self.assertEqual(title, '"No Error", testing alert for more than 2 comics with 1 title')

    def test_one_letter_diff(self):
        manhua = Types.Manhua
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

    def test_title_prefix_finds_full_title_same_type(self):
        with Session() as session:
            full_title = "I became the tyrant of a defense game"
            session.add(ComicDB(None, full_title, 20, com_type=Types.Manhwa))
            session.add(ComicDB(None, full_title, 20, com_type=Types.Novel))
            session.commit()

            matches = comics_by_title_prefix(
                "I became the tyrant",
                Types.Manhwa,
                session,
            )

            self.assertEqual(len(matches), 1)
            self.assertEqual(matches[0].get_titles()[0], full_title.capitalize())
            self.assertEqual(matches[0].com_type, Types.Manhwa)

    def test_title_prefix_ignores_leading_article_and_spacing(self):
        with Session() as session:
            full_title = "The holy emperor's grandson is a necromancer"
            session.add(ComicDB(None, full_title, 20, com_type=Types.Manhwa))
            session.add(ComicDB(None, "The holy emperor's grandsonis a necromancer", 20, com_type=Types.Novel))
            session.commit()

            matches = comics_by_title_prefix(
                "Holy emperor's grandson is a necr",
                Types.Manhwa,
                session,
            )

            self.assertEqual(len(matches), 1)
            self.assertEqual(matches[0].get_titles()[0], full_title.capitalize())

    def test_scrape_uses_unique_prefix_match_without_storing_truncated_alias(self):
        with Session() as session:
            full_title = "The max level hero has returned"
            existing = ComicDB(None, full_title, 20, com_type=Types.Manhwa)
            session.add(existing)
            session.commit()

            asyncio.run(register_comic(
                ScrapedComic(
                    chapter="Chapter 22",
                    title="The max level hero",
                    cover_url="https://example.com/cover.webp",
                    com_type="manhwa",
                    status="ongoing",
                ),
                Publishers.Asura,
                session,
            ))
            session.commit()

            comics = session.query(ComicDB).order_by(ComicDB.id).all()
            self.assertEqual(len(comics), 1)
            self.assertEqual(comics[0].current_chap, 22)
            self.assertEqual(comics[0].get_titles(), [full_title.capitalize()])

    def test_scrape_does_not_use_ambiguous_prefix_match(self):
        with Session() as session:
            session.add(ComicDB(None, "Return of the mount hua sect", 20, com_type=Types.Manhwa))
            session.add(ComicDB(None, "Return of the frozen player", 20, com_type=Types.Manhwa))
            session.commit()

            asyncio.run(register_comic(
                ScrapedComic(
                    chapter="Chapter 22",
                    title="Return of the",
                    cover_url="https://example.com/cover.webp",
                    com_type="manhwa",
                    status="ongoing",
                ),
                Publishers.Asura,
                session,
            ))
            session.commit()

            comics = session.query(ComicDB).order_by(ComicDB.id).all()
            self.assertEqual(len(comics), 3)
            self.assertEqual(comics[-1].get_titles(), ["Return of the"])


class TestCoverPriority(unittest.TestCase):
    def test_demonic_replaces_nelomanga_cover(self):
        comic = ComicDB(
            None,
            "Cover test",
            1,
            cover="https://nelomanga.example/cover.webp",
            published_in=[Publishers.Manganato],
        )

        self.assertTrue(
            _should_update_cover(
                comic,
                "https://demonic.example/cover.webp",
                Publishers.DemonicScans,
            )
        )

    def test_visible_demonic_cover_is_not_replaced(self):
        comic = ComicDB(
            None,
            "Cover test",
            1,
            cover="https://demonic.example/cover.webp",
            published_in=[Publishers.DemonicScans],
        )

        self.assertFalse(
            _should_update_cover(
                comic,
                "https://asura.example/cover.webp",
                Publishers.Asura,
            )
        )

    def test_invisible_cover_can_be_replaced_by_any_source(self):
        comic = ComicDB(
            None,
            "Cover test",
            1,
            cover="https://demonic.example/cover.webp",
            published_in=[Publishers.DemonicScans],
        )
        comic.cover_visible = False

        self.assertTrue(
            _should_update_cover(
                comic,
                "https://asura.example/cover.webp",
                Publishers.Asura,
            )
        )

    def test_nelomanga_does_not_replace_better_existing_source(self):
        comic = ComicDB(
            None,
            "Cover test",
            1,
            cover="https://asura.example/cover.webp",
            published_in=[Publishers.Asura, Publishers.Manganato],
        )

        self.assertFalse(
            _should_update_cover(
                comic,
                "https://nelomanga.example/cover.webp",
                Publishers.Manganato,
            )
        )


def main():
    unittest.main()


if __name__ == '__main__':
    unittest.main()
