import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import AddAppButton from '@/components/App/AddAppButton';

const createApp = vi.hoisted(() => vi.fn());
vi.mock('@/store/devApi', () => ({
  useInfoQuery: () => ({ data: { isSingleNodeService: false } }),
}));
vi.mock('@/store/generated', () => ({
  useCreateAppMutation: () => [createApp],
}));
afterEach(() => {
  cleanup();
  createApp.mockClear();
});

it('rejects invalid manual-sync URLs submitted with Enter', async () => {
  render(
    <>
      <div id="modals" />
      <AddAppButton secondary label="Sync manually" />
    </>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Sync manually' }));
  const input = document.getElementById('addAppUrlModal')!;
  fireEvent.change(input, { target: { value: 'invalid' } });
  fireEvent.keyDown(input, { key: 'Enter' });
  expect(createApp).not.toHaveBeenCalled();
  expect(screen.getByText('Please enter a valid URL')).toBeTruthy();
});
