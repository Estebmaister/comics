import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import {
  checkoutComic,
  dataFetch,
  delComic,
  isAbortError,
  rateComic,
  reportCoverVisibility,
  trackComic,
} from './ServerHelpers';
import { storeAccessToken } from './AuthHelpers';

beforeEach(() => {
  sessionStorage.clear();
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status: 403,
    json: () => Promise.resolve({ error: 'Insufficient privileges' }),
  }));
});

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test('dataFetch propagates abort signal to fetch', async () => {
  const abort = new AbortController();
  abort.abort();
  const fetchMock = vi.fn().mockRejectedValue(new DOMException('Aborted', 'AbortError'));
  vi.stubGlobal('fetch', fetchMock);

  await expect(dataFetch({
    from: 0,
    limit: 12,
    queryFilter: '',
    onlyTracked: false,
    onlyUnchecked: false,
    signal: abort.signal,
    server: 'https://api.example.test',
  })).rejects.toSatisfy((error: unknown) => isAbortError(error));

  expect(fetchMock).toHaveBeenCalledWith(
    expect.stringContaining('/comics?'),
    expect.objectContaining({ signal: abort.signal }),
  );
});

test('denied card mutations neither report success nor change local state', async () => {
  const setTrack = vi.fn();
  const setComic = vi.fn();
  const setCheck = vi.fn();
  const setViewedChap = vi.fn();
  const checkoutSuccess = vi.fn();
  const setDelete = vi.fn();
  const deleteSuccess = vi.fn();

  await expect(trackComic(false, 7, setTrack, 'https://api.example.test')).resolves.toBe(false);
  await expect(rateComic(4, 7, setComic, 'https://api.example.test')).resolves.toBe(false);
  await expect(
    checkoutComic(12, 7, setCheck, setViewedChap, checkoutSuccess, 'https://api.example.test'),
  ).resolves.toBe(false);
  await expect(
    delComic(7, setDelete, deleteSuccess, 'https://api.example.test'),
  ).resolves.toBe(false);
  await expect(
    reportCoverVisibility(7, 'cover.webp', false, 'https://api.example.test'),
  ).rejects.toThrow('Insufficient privileges');

  expect(setTrack).not.toHaveBeenCalled();
  expect(setComic).not.toHaveBeenCalled();
  expect(setCheck).not.toHaveBeenCalled();
  expect(setViewedChap).not.toHaveBeenCalled();
  expect(checkoutSuccess).not.toHaveBeenCalled();
  expect(setDelete).not.toHaveBeenCalled();
  expect(deleteSuccess).not.toHaveBeenCalled();

  for (const [, init] of vi.mocked(fetch).mock.calls) {
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
  }
});
