import { render, screen } from '@testing-library/react';
import { ComicsMainPage } from './MainPage';
import { COMIC_SEARCH_PLACEHOLDER } from '../constants';

jest.mock('react-router-dom', () => ({
  useSearchParams: () => [new URLSearchParams(), jest.fn()],
}), { virtual: true });

jest.mock('../../../util/ServerHelpers', () => ({
  dataFetch: jest.fn(),
}));

test(`renders "${COMIC_SEARCH_PLACEHOLDER}"`, () => {
  render(<ComicsMainPage />);
  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  expect(inputElement).toBeDefined();
});
