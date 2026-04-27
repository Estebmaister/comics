#!/usr/bin/env python3

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from typing import Iterable, List

import requests
from requests import Response
from requests.exceptions import RequestException, Timeout

from . import ComicDB, Session
from .repo import sync_json_backup_records

VISIBLE = "visible"
INVISIBLE = "invisible"
UNKNOWN = "unknown"

REQUEST_HEADERS = {
    "Accept": "image/avif,image/webp,image/apng,image/*,*/*;q=0.8",
    "User-Agent": (
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124 Safari/537.36"
    ),
}


@dataclass(frozen=True)
class CoverProbe:
    comic_id: int
    title: str
    cover: str


@dataclass(frozen=True)
class CoverProbeResult:
    comic_id: int
    title: str
    cover: str
    status: str
    reason: str


def _response_has_image_bytes(response: Response) -> bool:
    for chunk in response.iter_content(chunk_size=512):
        if chunk:
            return True
    return False


def probe_cover(
    probe: CoverProbe,
    *,
    timeout: float,
    retries: int,
) -> CoverProbeResult:
    attempts = max(1, retries + 1)
    last_reason = "not checked"
    for _ in range(attempts):
        try:
            with requests.get(
                probe.cover,
                headers=REQUEST_HEADERS,
                stream=True,
                timeout=timeout,
            ) as response:
                if response.status_code >= 400:
                    return CoverProbeResult(
                        probe.comic_id,
                        probe.title,
                        probe.cover,
                        INVISIBLE,
                        f"HTTP {response.status_code}",
                    )
                if _response_has_image_bytes(response):
                    return CoverProbeResult(
                        probe.comic_id,
                        probe.title,
                        probe.cover,
                        VISIBLE,
                        f"HTTP {response.status_code}",
                    )
                last_reason = f"HTTP {response.status_code} with empty body"
        except Timeout as error:
            last_reason = f"timeout: {error}"
        except RequestException as error:
            last_reason = f"request error: {error}"
            return CoverProbeResult(
                probe.comic_id,
                probe.title,
                probe.cover,
                UNKNOWN,
                last_reason,
            )

    return CoverProbeResult(
        probe.comic_id,
        probe.title,
        probe.cover,
        INVISIBLE,
        last_reason,
    )


def load_cover_probes(limit: int | None = None) -> List[CoverProbe]:
    with Session() as session:
        query = (
            session.query(ComicDB)
            .filter(ComicDB.cover != "")
            .order_by(ComicDB.id)
        )
        if limit is not None:
            query = query.limit(limit)
        return [
            CoverProbe(
                comic_id=int(comic.id),
                title=comic.get_titles()[0],
                cover=comic.cover,
            )
            for comic in query.all()
        ]


def run_cover_audit(
    probes: Iterable[CoverProbe],
    *,
    timeout: float,
    retries: int,
    concurrency: int,
) -> List[CoverProbeResult]:
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = [
            executor.submit(
                probe_cover,
                probe,
                timeout=timeout,
                retries=retries,
            )
            for probe in probes
        ]
        return [future.result() for future in as_completed(futures)]


def apply_visibility_results(
    results: Iterable[CoverProbeResult],
    *,
    include_unknown: bool,
) -> int:
    mark_statuses = {INVISIBLE}
    if include_unknown:
        mark_statuses.add(UNKNOWN)

    updated_records = []
    with Session() as session:
        for result in results:
            if result.status not in mark_statuses:
                continue
            comic = session.query(ComicDB).get(result.comic_id)
            if comic is None or comic.cover != result.cover:
                continue
            comic.cover_visible = False
            updated_records.append(comic.toJSON())

        session.commit()

    if updated_records:
        sync_json_backup_records(updated_records)
    return len(updated_records)


def _print_summary(results: List[CoverProbeResult], applied_count: int) -> None:
    counts = {
        VISIBLE: 0,
        INVISIBLE: 0,
        UNKNOWN: 0,
    }
    for result in results:
        counts[result.status] = counts.get(result.status, 0) + 1

    print(f"Visible covers: {counts[VISIBLE]}")
    print(f"Invisible covers: {counts[INVISIBLE]}")
    print(f"Unknown covers: {counts[UNKNOWN]}")
    print(f"Updated records: {applied_count}")

    for result in sorted(results, key=lambda item: (item.status, item.comic_id)):
        if result.status == VISIBLE:
            continue
        print(
            f"{result.status.upper()} {result.comic_id}: "
            f"{result.title} - {result.reason} - {result.cover}"
        )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Audit comic cover URLs and optionally mark failed covers invisible."
    )
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--include-unknown", action="store_true")
    parser.add_argument("--timeout", type=float, default=5.0)
    parser.add_argument("--retries", type=int, default=1)
    parser.add_argument("--concurrency", type=int, default=16)
    parser.add_argument("--limit", type=int)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    probes = load_cover_probes(args.limit)
    results = run_cover_audit(
        probes,
        timeout=args.timeout,
        retries=args.retries,
        concurrency=max(1, args.concurrency),
    )
    applied_count = 0
    if args.apply:
        applied_count = apply_visibility_results(
            results,
            include_unknown=args.include_unknown,
        )
    _print_summary(results, applied_count)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
