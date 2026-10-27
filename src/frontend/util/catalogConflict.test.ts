import { expect, test } from 'vitest';
import {
  mergePairFromConflict,
  parseCatalogConflictPayload,
} from './catalogConflict';

test('parseCatalogConflictPayload reads duplicate match fields', () => {
  const parsed = parseCatalogConflictPayload({
    code: 'duplicate_comic_match',
    message: 'Another catalog entry matches this comic\'s titles',
    comic_id: 34401,
    owner_comic_id: 35066,
    match_kind: 'title_prefix',
  }, 409, 34401);

  expect(parsed).toEqual({
    code: 'duplicate_comic_match',
    sourceComicId: 34401,
    ownerComicId: 35066,
    candidateIds: [],
    matchKind: 'title_prefix',
    message: 'Another catalog entry matches this comic\'s titles',
  });
});

test('mergePairFromConflict uses lowest id as base', () => {
  const pair = mergePairFromConflict({
    code: 'duplicate_comic_match',
    sourceComicId: 35066,
    ownerComicId: 34401,
    candidateIds: [],
    message: '',
  });
  expect(pair).toEqual({ baseId: 34401, mergingId: 35066 });
});
