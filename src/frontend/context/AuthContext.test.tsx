import { act, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { AuthProvider, useAuth } from './AuthContext';
import { AUTH_SESSION_EXPIRED_EVENT } from '../api/client';
import { storeAccessToken } from '../util/AuthHelpers';

const AuthProbe = () => {
  const { isAuthenticated, isAdmin, user } = useAuth();
  return (
    <output>
      {`${isAuthenticated ? 'authenticated' : 'anonymous'}:${isAdmin ? 'admin' : 'not-admin'}:${user?.role ?? 'no-role'}`}
    </output>
  );
};

beforeEach(() => {
  sessionStorage.clear();
});

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test('hydrates the persisted profile role and exposes isAdmin', async () => {
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({
      id: 'user-1',
      username: 'catalog-admin',
      email: 'admin@example.test',
      role: 'admin',
    }),
  }));

  render(
    <AuthProvider>
      <AuthProbe />
    </AuthProvider>,
  );

  await waitFor(() => {
    expect(screen.getByText('authenticated:admin:admin')).toBeInTheDocument();
  });

  const [, init] = vi.mocked(fetch).mock.calls[0];
  expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer admin-access-token');
});

test('clears in-memory auth when an authorized request receives 401', async () => {
  storeAccessToken('admin-access-token');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({
      id: 'user-1',
      username: 'catalog-admin',
      email: 'admin@example.test',
      role: 'admin',
    }),
  }));

  render(
    <AuthProvider>
      <AuthProbe />
    </AuthProvider>,
  );
  await waitFor(() => {
    expect(screen.getByText('authenticated:admin:admin')).toBeInTheDocument();
  });

  act(() => {
    window.dispatchEvent(new Event(AUTH_SESSION_EXPIRED_EVENT));
  });

  expect(screen.getByText('anonymous:not-admin:no-role')).toBeInTheDocument();
});
