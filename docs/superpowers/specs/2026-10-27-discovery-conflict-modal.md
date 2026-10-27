# Discovery conflict merge modal

## Problem

Admin catalog mutations (`PUT /comics/:id`) can fail when discovery finds another row (exact key, prefix, or tokens). The UI previously failed silently on rating and offered no guided merge path.

## UX

- On `409` with `discovery_key_conflict`, `duplicate_comic_match`, or `ambiguous_comic_match`, show a **toast** (“Catalog conflict”) and open a **modal** (shared `Modal` shell + `Modal.css` form patterns).
- Modal loads both comics via public `GET /comics/:id`, shows titles/chapter/rating/track, and preselects merge **base = lower ID** (repo convention).
- **Merge & continue** calls existing `PATCH /comics/{base}/{merging}`; refreshes catalog; retries the original mutation when provided.
- **Not now** dismisses without merge.

## API

- Fuzzy duplicate checks return `DuplicateComicMatchError` → `409` + `duplicate_comic_match` with `comic_id`, `owner_comic_id`, `match_kind`.
- `discovery_key_conflict` responses include `comic_id` for symmetry.

## Scope

- Wired for admin card actions: rate, track, checkout, edit.
- Provider: `DiscoveryConflictProvider` on comics main page (inside `ToastProvider`).
