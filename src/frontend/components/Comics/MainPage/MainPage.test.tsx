import { act, fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, vi } from 'vitest';
import { ComicsMainPage } from './MainPage';
import { AuthProvider } from '../../../context/AuthContext';
import { appBasename } from '../../../util/RouterBasename';
import { COMIC_SEARCH_PLACEHOLDER, REFRESH_INTERVAL, SEARCH_DEBOUNCE_MS } from '../constants';
import { dataFetch } from '../../../util/ServerHelpers';

vi.mock('../../../util/ServerHelpers', () => ({
  dataFetch: vi.fn(),
  ComicListFetchError: class ComicListFetchError extends Error {
    constructor(message: string, public readonly kind: 'network' | 'server' = 'network') {
      super(message);
    }
  },
  isAbortError: (error: unknown) => (
    error !== null
    && typeof error === 'object'
    && 'name' in error
    && (error as { name?: string }).name === 'AbortError'
  ),
}));

const dataFetchMock = vi.mocked(dataFetch);
const emptyComicsResult = {
  comics: [],
  pagination: {
    total: 0,
    totalPages: 1,
    currentPage: 1,
  },
};

const renderPage = () => render(
  <MemoryRouter basename={appBasename} initialEntries={[appBasename ? `${appBasename}/` : '/']}>
    <AuthProvider>
      <ComicsMainPage />
    </AuthProvider>
  </MemoryRouter>
);

beforeEach(() => {
  vi.useFakeTimers();
  dataFetchMock.mockReset();
  dataFetchMock.mockResolvedValue(emptyComicsResult);
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    value: 'visible',
  });
});

afterEach(() => {
  vi.useRealTimers();
});

const flushPromises = () => act(async () => {
  await Promise.resolve();
});

test(`renders "${COMIC_SEARCH_PLACEHOLDER}"`, async () => {
  renderPage();
  await flushPromises();

  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  expect(inputElement).toBeDefined();
});

test('debounces search requests from the toolbar input', async () => {
  renderPage();
  await flushPromises();

  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  fireEvent.change(inputElement, { target: { value: 'solo' } });

  act(() => {
    vi.advanceTimersByTime(SEARCH_DEBOUNCE_MS - 1);
  });
  expect(dataFetchMock).toHaveBeenCalledTimes(1);

  await act(async () => {
    vi.advanceTimersByTime(1);
    await Promise.resolve();
  });

  expect(dataFetchMock.mock.calls.some(([args]) => args.queryFilter === 'solo')).toBe(true);
});

test('aborts stale list requests when filters change', async () => {
  let firstSignal: AbortSignal | undefined;
  dataFetchMock
    .mockImplementationOnce(({ signal }) => {
      firstSignal = signal;
      return new Promise<never>(() => undefined);
    })
    .mockResolvedValue(emptyComicsResult);

  renderPage();

  expect(firstSignal).toBeDefined();
  fireEvent.click(screen.getByRole('button', { name: /tracked/i }));
  await flushPromises();

  expect(firstSignal?.aborted).toBe(true);
  expect(dataFetchMock).toHaveBeenCalledTimes(2);
});

test('applies rating filter and sort params from sort/filter modal', async () => {
  renderPage();
  await flushPromises();

  fireEvent.click(screen.getByRole('button', { name: /open sort and filter options/i }));
  act(() => {
    vi.runOnlyPendingTimers();
  });
  await flushPromises();

  const ratingMinInput = screen.getByLabelText(/rating min/i);
  fireEvent.change(ratingMinInput, { target: { value: '3' } });

  const sortBySelect = screen.getByLabelText(/sort by/i);
  fireEvent.change(sortBySelect, { target: { value: 'rating' } });

  fireEvent.click(screen.getByRole('button', { name: /apply/i }));
  act(() => {
    vi.runOnlyPendingTimers();
  });
  await flushPromises();

  expect(
    dataFetchMock.mock.calls.some(([args]) => args.ratingMin === 3 && args.sortBy === 'rating')
  ).toBe(true);
});

test('skips automatic refresh while the tab is hidden', async () => {
  renderPage();
  await flushPromises();
  expect(dataFetchMock).toHaveBeenCalledTimes(1);

  act(() => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      value: 'hidden',
    });
    document.dispatchEvent(new Event('visibilitychange'));
  });

  act(() => {
    vi.advanceTimersByTime(REFRESH_INTERVAL);
  });

  expect(dataFetchMock).toHaveBeenCalledTimes(1);
});

test('does not overlap automatic refresh requests', () => {
  dataFetchMock.mockImplementationOnce(() => new Promise<never>(() => undefined));

  renderPage();
  expect(dataFetchMock).toHaveBeenCalledTimes(1);

  act(() => {
    vi.advanceTimersByTime(REFRESH_INTERVAL);
  });

  expect(dataFetchMock).toHaveBeenCalledTimes(1);
});
