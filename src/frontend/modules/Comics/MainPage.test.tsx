import { render, screen } from '@testing-library/react';
import { BrowserRouter as Router } from 'react-router-dom';
import { ComicsMainPage } from './MainPage';
import { COMIC_SEARCH_PLACEHOLDER } from './constants';

test(`renders "${COMIC_SEARCH_PLACEHOLDER}"`, () => {
  render(<Router> <ComicsMainPage /> </Router>);
  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  expect(inputElement).toBeDefined();
});
