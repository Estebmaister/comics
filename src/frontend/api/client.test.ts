import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import {
  AUTH_SESSION_EXPIRED_EVENT,
  apiErrorMessage,
  authorizedFetch,
} from './client';
import { readAccessToken, storeAccessToken } from '../util/AuthHelpers';

describe('authorizedFetch', () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    sessionStorage.clear();
    vi.unstubAllGlobals();
  });

  test('attaches the stored bearer token and JSON headers', async () => {
    storeAccessToken('admin-access-token');
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      status: 200,
    } as Response);

    await authorizedFetch(
      '/comics/7',
      {
        method: 'PUT',
        body: JSON.stringify({ track: true }),
        headers: { 'X-Request-Test': 'yes' },
      },
      'https://api.example.test',
    );

    expect(fetch).toHaveBeenCalledTimes(1);
    const [url, init] = vi.mocked(fetch).mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(url).toBe('https://api.example.test/comics/7');
    expect(headers.get('Authorization')).toBe('Bearer admin-access-token');
    expect(headers.get('Accept')).toBe('application/json');
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(headers.get('X-Request-Test')).toBe('yes');
    expect(init?.credentials).not.toBe('include');
  });

  test('clears the session and emits expiration on 401', async () => {
    storeAccessToken('expired-token');
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 401,
    } as Response);
    const listener = vi.fn();
    window.addEventListener(AUTH_SESSION_EXPIRED_EVENT, listener);

    try {
      await authorizedFetch('/comics/7', { method: 'DELETE' }, 'https://api.example.test');
    } finally {
      window.removeEventListener(AUTH_SESSION_EXPIRED_EVENT, listener);
    }

    expect(readAccessToken()).toBeUndefined();
    expect(listener).toHaveBeenCalledTimes(1);
  });

  test('keeps a valid session on 403', async () => {
    storeAccessToken('user-access-token');
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 403,
    } as Response);

    await authorizedFetch('/comics/7', { method: 'DELETE' }, 'https://api.example.test');

    expect(readAccessToken()).toBe('user-access-token');
  });

  test('normalizes message and error response bodies', () => {
    expect(apiErrorMessage({ message: 'Conflict' }, 'Fallback')).toBe('Conflict');
    expect(apiErrorMessage({ error: 'Insufficient privileges' }, 'Fallback'))
      .toBe('Insufficient privileges');
    expect(apiErrorMessage('plain failure', 'Fallback')).toBe('plain failure');
    expect(apiErrorMessage(null, 'Fallback')).toBe('Fallback');
  });
});
