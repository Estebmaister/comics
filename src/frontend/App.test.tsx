import { render, screen } from '@testing-library/react';
import {App, COMIC_SEARCH_PLACEHOLDER} from './App';

test(`renders "${COMIC_SEARCH_PLACEHOLDER}"`, () => {
  render(<App />);
  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  expect(inputElement).toBeDefined();
});
