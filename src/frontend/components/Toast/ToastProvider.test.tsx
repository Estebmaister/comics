import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { ToastProvider, useToast } from './ToastProvider';

const ToastHarness = () => {
  const toast = useToast();

  return (
    <>
      <button onClick={() => toast.success({ title: 'Saved', description: 'Comic updated.' })}>
        Success
      </button>
      <button onClick={() => toast.error({ title: 'Failed', description: 'Merge failed.' })}>
        Error
      </button>
      <button onClick={() => {
        toast.success({ title: 'First', description: 'One' });
        toast.info({ title: 'Second', description: 'Two' });
      }}>
        Stack
      </button>
    </>
  );
};

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  act(() => {
    vi.runOnlyPendingTimers();
  });
  vi.useRealTimers();
});

test('renders stacked notifications and allows dismiss', () => {
  render(
    <ToastProvider>
      <ToastHarness />
    </ToastProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: 'Stack' }));

  expect(screen.getByText('First')).toBeTruthy();
  expect(screen.getByText('Second')).toBeTruthy();

  act(() => {
    fireEvent.click(screen.getByRole('button', { name: /dismiss notification: first/i }));
  });
  expect(screen.queryByText('First')).toBeNull();
  expect(screen.getByText('Second')).toBeTruthy();
});

test('auto hides success and error notifications on their timers', () => {
  render(
    <ToastProvider>
      <ToastHarness />
    </ToastProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: 'Success' }));
  expect(screen.getByText('Saved')).toBeTruthy();

  act(() => {
    vi.advanceTimersByTime(3600);
  });
  expect(screen.queryByText('Saved')).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'Error' }));
  expect(screen.getByText('Failed')).toBeTruthy();

  act(() => {
    vi.advanceTimersByTime(6100);
  });
  expect(screen.queryByText('Failed')).toBeNull();
});
