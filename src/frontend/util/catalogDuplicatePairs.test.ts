import { expect, test } from 'vitest';
import { findCatalogDuplicatePairs } from './catalogDuplicatePairs';
import type { Comic } from '../components/Comics/types';

const comic = (id: number, title: string, chapter = 32): Comic => ({
  id,
  titles: [title],
  cover: '',
  author: '',
  current_chap: chapter,
  viewed_chap: chapter,
  track: true,
  status: 2,
  com_type: 3,
  genres: [],
  published_in: [],
});

test('findCatalogDuplicatePairs detects truncated vs full title on same page', () => {
  const pairs = findCatalogDuplicatePairs([
    comic(34401, "I may be a beginner, but i'm actually"),
    comic(35066, "I may be a beginner, but i'm actually a level 99 overpowered player"),
  ]);
  expect(pairs).toHaveLength(1);
  expect(pairs[0]).toMatchObject({
    baseId: 34401,
    mergingId: 35066,
    matchKind: 'title_prefix',
  });
});
