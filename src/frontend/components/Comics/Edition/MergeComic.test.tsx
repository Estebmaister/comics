import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import MergeComic from './MergeComic';
import { ToastProvider } from '../../Toast/ToastProvider';

test('keeps merge draft values when the modal is closed and reopened', async () => {
  render(
    <ToastProvider>
      <MergeComic />
    </ToastProvider>
  );

  const mergeButton = screen.getByRole('button', { name: /merge comics/i });

  mergeButton.focus();
  fireEvent.click(mergeButton);
  expect(screen.getByRole('dialog')).toBeTruthy();
  await waitFor(() => {
    expect(document.activeElement).toBe(screen.getByLabelText(/baseid/i));
  });

  fireEvent.change(screen.getByLabelText(/baseid/i), { target: { value: '579' } });
  fireEvent.click(screen.getByRole('button', { name: /close/i }));
  expect(document.activeElement).toBe(mergeButton);

  fireEvent.click(mergeButton);

  const baseInput = screen.getByLabelText(/baseid/i) as HTMLInputElement;
  expect(baseInput.value).toBe('579');
  await waitFor(() => {
    expect(document.activeElement).toBe(baseInput);
  });
});
