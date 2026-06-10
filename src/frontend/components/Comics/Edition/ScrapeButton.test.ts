import { afterEach, describe, expect, test, vi } from 'vitest';
import { scrape } from './ScrapeButton';

describe('scrape', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test('returns false for non-ok responses', async () => {
    const setShowLoader = vi.fn();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ message: 'Python scrape backend is not configured' }),
    }));

    await expect(scrape(setShowLoader, 'http://localhost:8081')).resolves.toBe(false);
    expect(setShowLoader).toHaveBeenNthCalledWith(1, true);
    expect(setShowLoader).toHaveBeenLastCalledWith(false);
  });

  test('returns true for ok responses', async () => {
    const setShowLoader = vi.fn();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    }));

    await expect(scrape(setShowLoader, 'http://localhost:8081')).resolves.toBe(true);
  });
});
