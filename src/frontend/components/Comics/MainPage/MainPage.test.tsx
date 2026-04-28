import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { ComicsMainPage } from './MainPage';
import { COMIC_SEARCH_PLACEHOLDER } from '../constants';

vi.mock('react-router-dom', () => ({
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
}));

vi.mock('../../../util/ServerHelpers', () => ({
  dataFetch: vi.fn(),
}));

test(`renders "${COMIC_SEARCH_PLACEHOLDER}"`, () => {
  render(<ComicsMainPage />);
  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  expect(inputElement).toBeDefined();
});
