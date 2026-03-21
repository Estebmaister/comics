import { fireEvent, render, screen, within } from '@testing-library/react';
import ComicCard from './ComicCard';
import type { Comic } from '../types';
import { ToastProvider } from '../../Toast/ToastProvider';

jest.mock('../../../hooks/useComicActions', () => ({
  useComicActions: () => ({
    handleCheckout: jest.fn(),
    handleTrackToggle: jest.fn(),
    handleDelete: jest.fn(),
  }),
}));

const comicFixture: Comic = {
  id: 15816,
  titles: ['Rise from the bottom'],
  cover: 'https://example.com/cover.jpg',
  author: 'DemonicSca',
  current_chap: 62,
  viewed_chap: 48,
  track: true,
  status: 2,
  com_type: 3,
  genres: [0, 1],
  published_in: [0, 1],
  description: '',
};

beforeAll(() => {
  Object.assign(navigator, {
    clipboard: {
      writeText: jest.fn(),
    },
  });
});

test('renders overlay controls and footer action lane', () => {
  render(
    <ToastProvider>
      <ComicCard comic={comicFixture} />
    </ToastProvider>
  );

  expect(screen.getByRole('button', { name: /delete rise from the bottom/i })).toBeTruthy();
  expect(screen.getByRole('button', { name: /edit/i })).toBeTruthy();
  expect(screen.getByRole('button', { name: /copy comic id 15816/i })).toBeTruthy();

  const footerActions = screen.getByTestId('comic-footer-actions');
  expect(within(footerActions).getByRole('button', { name: /checkout/i })).toBeTruthy();
  expect(within(footerActions).getByRole('button', { name: /untrack/i })).toBeTruthy();
});

test('copies the comic id and shows themed fallback when the cover fails', async () => {
  render(
    <ToastProvider>
      <ComicCard comic={comicFixture} />
    </ToastProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: /copy comic id 15816/i }));
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith('15816');

  fireEvent.error(screen.getByRole('img', { name: /rise from the bottom/i }));
  expect(screen.getByTestId('poster-fallback')).toBeTruthy();
  expect(screen.getByText(/cover unavailable/i)).toBeTruthy();
});
