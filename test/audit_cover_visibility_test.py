import unittest
from unittest.mock import patch

from requests.exceptions import RequestException, Timeout

from src.db.audit_cover_visibility import (
    INVISIBLE,
    UNKNOWN,
    VISIBLE,
    CoverProbe,
    probe_cover,
)


class FakeResponse:
    def __init__(self, status_code=200, chunks=None):
        self.status_code = status_code
        self._chunks = chunks if chunks is not None else [b"image"]

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False

    def iter_content(self, chunk_size=512):
        yield from self._chunks


class TestCoverAuditProbe(unittest.TestCase):
    def test_probe_cover_marks_successful_image_visible(self):
        probe = CoverProbe(1, "Readable", "https://example.com/cover.webp")
        with patch(
            "src.db.audit_cover_visibility.requests.get",
            return_value=FakeResponse(200, [b"image"]),
        ):
            result = probe_cover(probe, timeout=0.1, retries=0)

        self.assertEqual(result.status, VISIBLE)

    def test_probe_cover_marks_http_error_invisible(self):
        probe = CoverProbe(1, "Missing", "https://example.com/cover.webp")
        with patch(
            "src.db.audit_cover_visibility.requests.get",
            return_value=FakeResponse(404, [b""]),
        ):
            result = probe_cover(probe, timeout=0.1, retries=0)

        self.assertEqual(result.status, INVISIBLE)
        self.assertIn("HTTP 404", result.reason)

    def test_probe_cover_marks_timeout_invisible_after_retries(self):
        probe = CoverProbe(1, "Slow", "https://example.com/cover.webp")
        with patch(
            "src.db.audit_cover_visibility.requests.get",
            side_effect=Timeout("too slow"),
        ):
            result = probe_cover(probe, timeout=0.1, retries=1)

        self.assertEqual(result.status, INVISIBLE)

    def test_probe_cover_marks_ambiguous_request_error_unknown(self):
        probe = CoverProbe(1, "Ambiguous", "https://example.com/cover.webp")
        with patch(
            "src.db.audit_cover_visibility.requests.get",
            side_effect=RequestException("connection reset"),
        ):
            result = probe_cover(probe, timeout=0.1, retries=0)

        self.assertEqual(result.status, UNKNOWN)


if __name__ == "__main__":
    unittest.main()
