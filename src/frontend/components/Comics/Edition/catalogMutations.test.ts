import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { storeAccessToken } from '../../../util/AuthHelpers';
import type { Comic, CreateComicFormState } from '../types';
import { createComic } from './CreateComic';
import { editComic } from './EditComic';
import { mergeComic } from './MergeComic';

const createFixture: CreateComicFormState = {
  title: 'Boundary test',
  track: false,
  current_chap: 1,
  viewed_chap: 0,
  cover: 'cover.webp',
  description: '',
  author: '',
  com_type: 3,
  status: 0,
  published_in: [],
  genres: [],
};

const comicFixture: Comic = {
  id: 7,
  titles: ['Boundary test'],
  track: false,
  current_chap: 1,
  viewed_chap: 0,
  cover: 'cover.webp',
  author: '',
  com_type: 3,
  status: 0,
  published_in: [],
  genres: [],
};

beforeEach(() => {
  sessionStorage.clear();
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status: 403,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve({ error: 'Insufficient privileges' }),
    text: () => Promise.resolve('Insufficient privileges'),
  }));
});

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test('create edit and merge expose denied responses and send the bearer token', async () => {
  await expect(createComic(createFixture, 'https://api.example.test'))
    .resolves.toBe('Insufficient privileges');
  await expect(editComic(comicFixture, 'https://api.example.test'))
    .resolves.toEqual([undefined, 'Insufficient privileges']);
  await expect(mergeComic(7, 8, 'https://api.example.test'))
    .resolves.toBe('Insufficient privileges');

  expect(fetch).toHaveBeenCalledTimes(3);
  for (const [, init] of vi.mocked(fetch).mock.calls) {
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
  }
});
