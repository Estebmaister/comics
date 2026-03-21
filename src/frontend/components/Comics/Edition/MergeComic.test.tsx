import { fireEvent, render, screen } from '@testing-library/react';
import MergeComic from './MergeComic';
import { ToastProvider } from '../../Toast/ToastProvider';

beforeAll(() => {
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true,
    value: function showModal() {
      this.setAttribute('open', '');
    },
  });

  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true,
    value: function close() {
      this.removeAttribute('open');
    },
  });
});

test('keeps merge draft values when the modal is closed and reopened', () => {
  render(
    <ToastProvider>
      <MergeComic />
    </ToastProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: /merge comics/i }));
  fireEvent.change(screen.getByLabelText(/baseid/i), { target: { value: '579' } });
  fireEvent.click(screen.getByRole('button', { name: /close/i }));

  fireEvent.click(screen.getByRole('button', { name: /merge comics/i }));

  const baseInput = screen.getByLabelText(/baseid/i) as HTMLInputElement;
  expect(baseInput.value).toBe('579');
});
