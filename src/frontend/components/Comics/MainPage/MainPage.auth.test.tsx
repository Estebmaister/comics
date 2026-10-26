import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { ToastProvider } from '../../Toast/ToastProvider';
import { ComicsMainPage } from './MainPage';
import { dataFetch } from '../../../util/ServerHelpers';

const authState = vi.hoisted(() => ({
  user: null as null | { username: string; role: string },
  accessToken: undefined as string | undefined,
  isAuthenticated: false,
  isAdmin: false,
  login: vi.fn(),
  signup: vi.fn(),
  logout: vi.fn(),
  refreshProfile: vi.fn(),
}));

vi.mock('../../../context/AuthContext', () => ({
  useAuth: () => authState,
}));

vi.mock('../../../util/ServerHelpers', () => ({
  dataFetch: vi.fn(),
  ComicListFetchError: class ComicListFetchError extends Error {
    constructor(message: string, public readonly kind: 'network' | 'server' = 'network') {
      super(message);
    }
  },
  isAbortError: () => false,
}));

const renderPage = () => render(
  <ToastProvider>
    <MemoryRouter>
      <ComicsMainPage />
    </MemoryRouter>
  </ToastProvider>,
);

beforeEach(() => {
  authState.user = null;
  authState.accessToken = undefined;
  authState.isAuthenticated = false;
  authState.isAdmin = false;
  vi.mocked(dataFetch).mockResolvedValue({
    comics: [],
    pagination: { total: 0, totalPages: 1, currentPage: 1 },
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

test('shows catalog utilities only after an admin role is present', async () => {
  const view = renderPage();
  await waitFor(() => expect(dataFetch).toHaveBeenCalled());
  expect(screen.queryByRole('button', { name: /open comic utilities/i })).not.toBeInTheDocument();

  authState.user = { username: 'reader', role: 'user' };
  authState.accessToken = 'user-token';
  authState.isAuthenticated = true;
  view.rerender(
    <ToastProvider>
      <MemoryRouter>
        <ComicsMainPage />
      </MemoryRouter>
    </ToastProvider>,
  );
  expect(screen.queryByRole('button', { name: /open comic utilities/i })).not.toBeInTheDocument();

  authState.user = { username: 'catalog-admin', role: 'admin' };
  authState.accessToken = 'admin-token';
  authState.isAdmin = true;
  view.rerender(
    <ToastProvider>
      <MemoryRouter>
        <ComicsMainPage />
      </MemoryRouter>
    </ToastProvider>,
  );
  expect(screen.getByRole('button', { name: /open comic utilities/i })).toBeInTheDocument();
});
