import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { toast } from 'sonner';
import { QuietCopyButton } from './QuietCopyButton';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
beforeEach(() => {
  vi.useFakeTimers();
  vi.clearAllMocks();
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
  });
});
afterEach(() => {
  cleanup();
  vi.clearAllTimers();
  vi.useRealTimers();
});
it('confirms successful onboarding copies on the button without a toast', async () => {
  render(<QuietCopyButton content="npm install inngest" />);
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: 'Copy' })),
  );
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
    'npm install inngest',
  );
  expect(screen.getByText('Copied!')).toBeTruthy();
  expect(toast.success).not.toHaveBeenCalled();
  act(() => vi.advanceTimersByTime(1000));
  expect(screen.getByText('Copy')).toBeTruthy();
});
it('reports a failed onboarding copy without false success or a banner', async () => {
  vi.mocked(navigator.clipboard.writeText).mockRejectedValueOnce(
    new Error('denied'),
  );
  render(<QuietCopyButton content="configuration" />);
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: 'Copy' })),
  );
  expect(screen.getByRole('status').textContent).toContain('Couldn’t copy');
  expect(screen.queryByText('Copied!')).toBeNull();
  expect(toast.success).not.toHaveBeenCalled();
  expect(toast.error).not.toHaveBeenCalled();
});
