import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { storeAccessToken } from '../../../util/AuthHelpers';
import { scrape } from './ScrapeButton';

describe('scrape', () => {
  beforeEach(() => {
    sessionStorage.clear();
    storeAccessToken('admin-access-token');
  });

  afterEach(() => {
    sessionStorage.clear();
    vi.unstubAllGlobals();
  });

  test('returns not ok for non-ok responses', async () => {
    const setShowLoader = vi.fn();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ message: 'Python scrape backend is not configured' }),
    }));

    await expect(scrape(setShowLoader, 'http://localhost:8081')).resolves.toEqual({
      ok: false,
      busy: false,
    });
    expect(setShowLoader).toHaveBeenNthCalledWith(1, true);
    expect(setShowLoader).toHaveBeenLastCalledWith(false);
  });

  test('returns busy for 409 conflict', async () => {
    const setShowLoader = vi.fn();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      json: () => Promise.resolve({ message: 'scrape already in progress', source: 'automatic' }),
    }));

    await expect(scrape(setShowLoader, 'http://localhost:8081')).resolves.toEqual({
      ok: false,
      busy: true,
      source: 'automatic',
    });
  });

  test('returns ok for successful responses', async () => {
    const setShowLoader = vi.fn();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ message: 'success' }),
    }));

    await expect(scrape(setShowLoader, 'http://localhost:8081')).resolves.toEqual({
      ok: true,
      busy: false,
    });

    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
  });
});
