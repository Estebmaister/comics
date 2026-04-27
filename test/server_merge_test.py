import unittest
from unittest.mock import patch

import server as server_module


COMIC_PAYLOAD = {
    "id": 1,
    "titles": ["Base title"],
    "current_chap": 12,
    "cover": "https://example.com/cover.webp",
    "cover_visible": True,
    "last_update": "2026-01-01T00:00:00+00:00",
    "com_type": 3,
    "status": 2,
    "published_in": [17],
    "genres": [0],
    "description": "",
    "author": "",
    "track": False,
    "viewed_chap": 12,
    "rating": 0,
    "deleted": False,
}


class TestMergeEndpoint(unittest.TestCase):
    def setUp(self):
        self.client = server_module.server.test_client()
        self.origin = "https://localhost:3000"

    @patch.object(server_module, "merge_comics")
    def test_merge_route_accepts_patch_and_put_without_trailing_slash(self, merge_mock):
        merge_mock.return_value = (COMIC_PAYLOAD, None, 200)

        for method in ("patch", "put"):
            with self.subTest(method=method):
                response = getattr(self.client, method)(
                    "/comics/1/2",
                    headers={"Origin": self.origin},
                )

                self.assertEqual(response.status_code, 200)
                self.assertEqual(
                    response.headers.get("Access-Control-Allow-Origin"),
                    self.origin,
                )

    @patch.object(server_module, "merge_comics")
    def test_merge_route_preserves_cors_headers_on_unexpected_error(self, merge_mock):
        merge_mock.side_effect = RuntimeError("boom")

        response = self.client.patch(
            "/comics/1/2",
            headers={"Origin": self.origin},
        )

        self.assertEqual(response.status_code, 500)
        self.assertEqual(
            response.headers.get("Access-Control-Allow-Origin"),
            self.origin,
        )
        self.assertEqual(
            response.get_json(),
            {"message": "Internal server error while handling request"},
        )

    @patch.object(server_module, "update_cover_visibility_by_id")
    def test_cover_visibility_route_accepts_patch(self, visibility_mock):
        visibility_mock.return_value = {
            **COMIC_PAYLOAD,
            "cover_visible": False,
        }

        response = self.client.patch(
            "/comics/1/cover-visibility",
            json={
                "cover": COMIC_PAYLOAD["cover"],
                "cover_visible": False,
            },
            headers={"Origin": self.origin},
        )

        self.assertEqual(response.status_code, 200)
        visibility_mock.assert_called_once_with(
            1,
            COMIC_PAYLOAD["cover"],
            False,
        )
        self.assertEqual(response.get_json()["cover_visible"], False)

    @patch.object(server_module, "update_cover_visibility_by_id")
    def test_cover_visibility_route_rejects_invalid_payload(self, visibility_mock):
        response = self.client.patch(
            "/comics/1/cover-visibility",
            json={"cover": COMIC_PAYLOAD["cover"]},
            headers={"Origin": self.origin},
        )

        self.assertEqual(response.status_code, 400)
        visibility_mock.assert_not_called()


if __name__ == "__main__":
    unittest.main()
