import config from '../util/Config';
import { clearAuthStorage, readAccessToken } from '../util/AuthHelpers';

export const AUTH_SESSION_EXPIRED_EVENT = 'comics:auth-session-expired';

export const apiErrorMessage = (payload: unknown, fallback: string): string => {
  if (typeof payload === 'string' && payload.trim() !== '') {
    return payload;
  }
  if (payload !== null && typeof payload === 'object') {
    const body = payload as { message?: unknown; error?: unknown };
    if (typeof body.message === 'string' && body.message.trim() !== '') {
      return body.message;
    }
    if (typeof body.error === 'string' && body.error.trim() !== '') {
      return body.error;
    }
  }
  return fallback;
};

export const authorizedFetch = async (
  path: string,
  init: RequestInit = {},
  server = config.SERVER,
): Promise<Response> => {
  const headers = new Headers(init.headers ?? {});
  headers.set('Accept', 'application/json');
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const accessToken = readAccessToken();
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  const response = await fetch(`${server}${path}`, {
    ...init,
    headers,
  });

  if (response.status === 401) {
    clearAuthStorage();
    window.dispatchEvent(new Event(AUTH_SESSION_EXPIRED_EVENT));
  }

  return response;
};
