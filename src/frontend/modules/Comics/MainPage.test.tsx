import { render, screen } from '@testing-library/react';
import {BrowserRouter as Router} from 'react-router-dom';
import {ComicsMainPage, COMIC_SEARCH_PLACEHOLDER} from './MainPage';

test(`renders "${COMIC_SEARCH_PLACEHOLDER}"`, () => {
  render(<Router> <ComicsMainPage /> </Router>);
  const inputElement = screen.getByPlaceholderText(COMIC_SEARCH_PLACEHOLDER);
  expect(inputElement).toBeDefined();
});
