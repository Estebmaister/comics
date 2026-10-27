import { afterEach, describe, expect, test, vi } from 'vitest';
import {
  buildCatalogQueryKey,
  fetchCatalogQuery,
  invalidateCatalogQueryCache,
  peekCatalogQueryCache,
  resetCatalogQueryCacheForTests,
  scheduleAdjacentCatalogPrefetch,
} from './catalogQueryCache';
import type { ComicListFetchResult } from '../../../util/ServerHelpers';

const baseArgs = {
  from: 0,
  limit: 12,
  queryFilter: '',
  onlyTracked: false,
  onlyUnchecked: false,
};

const sampleResult: ComicListFetchResult = {
  comics: [{ id: 1, titles: ['A'], current_chap: 1, cover: '', last_update: 0, com_type: 0, status: 0, published_in: [], genres: [], description: '', author: '', track: false, viewed_chap: 0, rating: 0, deleted: false, cover_visible: true }],
  pagination: { total: 1, totalPages: 1, currentPage: 1 },
};

afterEach(() => {
  resetCatalogQueryCacheForTests();
});

test('buildCatalogQueryKey normalizes filter whitespace', () => {
  const a = buildCatalogQueryKey({ ...baseArgs, queryFilter: ' solo ' });
  const b = buildCatalogQueryKey({ ...baseArgs, queryFilter: 'solo' });
  expect(a).toBe(b);
});

test('fetchCatalogQuery does not dedupe signed in-flight catalog loads', async () => {
  const fetcher = vi.fn()
    .mockRejectedValueOnce(new DOMException('Aborted', 'AbortError'))
    .mockResolvedValueOnce(sampleResult);
  const key = buildCatalogQueryKey(baseArgs);
  const signal = new AbortController().signal;

  await expect(fetchCatalogQuery(key, { ...baseArgs, signal }, fetcher)).rejects.toMatchObject({ name: 'AbortError' });
  const result = await fetchCatalogQuery(key, { ...baseArgs, signal: new AbortController().signal }, fetcher);
  expect(result).toEqual(sampleResult);
  expect(fetcher).toHaveBeenCalledTimes(2);
});

test('fetchCatalogQuery deduplicates in-flight prefetch requests', async () => {
  const fetcher = vi.fn().mockImplementation(() => new Promise<ComicListFetchResult>((resolve) => {
    setTimeout(() => resolve(sampleResult), 20);
  }));
  const key = buildCatalogQueryKey(baseArgs);

  const [first, second] = await Promise.all([
    fetchCatalogQuery(key, baseArgs, fetcher),
    fetchCatalogQuery(key, baseArgs, fetcher),
  ]);

  expect(first).toEqual(sampleResult);
  expect(second).toEqual(sampleResult);
  expect(fetcher).toHaveBeenCalledTimes(1);
});

test('failed fetch does not poison cache', async () => {
  const fetcher = vi.fn()
    .mockRejectedValueOnce(new Error('boom'))
    .mockResolvedValueOnce(sampleResult);
  const key = buildCatalogQueryKey(baseArgs);

  await expect(fetchCatalogQuery(key, baseArgs, fetcher)).rejects.toThrow('boom');
  expect(peekCatalogQueryCache(key)).toBeUndefined();

  const result = await fetchCatalogQuery(key, baseArgs, fetcher);
  expect(result).toEqual(sampleResult);
  expect(fetcher).toHaveBeenCalledTimes(2);
});

test('invalidateCatalogQueryCache clears hits and in-flight map', async () => {
  const fetcher = vi.fn().mockResolvedValue(sampleResult);
  const key = buildCatalogQueryKey(baseArgs);
  await fetchCatalogQuery(key, baseArgs, fetcher);
  expect(peekCatalogQueryCache(key)).toEqual(sampleResult);

  invalidateCatalogQueryCache();
  expect(peekCatalogQueryCache(key)).toBeUndefined();
});

test('scheduleAdjacentCatalogPrefetch warms neighbor pages', async () => {
  const fetcher = vi.fn().mockResolvedValue(sampleResult);
  const idle = vi.fn((cb: IdleRequestCallback) => {
    cb({ didTimeout: false, timeRemaining: () => 50 } as IdleDeadline);
  });
  vi.stubGlobal('requestIdleCallback', idle);

  scheduleAdjacentCatalogPrefetch({ ...baseArgs, from: 12 }, fetcher);
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => setTimeout(resolve, 0));

  const offsets = fetcher.mock.calls.map(([args]) => args.from);
  expect(offsets).toContain(0);
  expect(offsets).toContain(24);
});
