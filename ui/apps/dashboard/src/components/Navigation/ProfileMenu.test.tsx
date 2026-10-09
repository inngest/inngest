// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

import { ProfileMenu } from './ProfileMenu';

const navigate = vi.hoisted(() => vi.fn());
vi.mock('@tanstack/react-router', () => ({ useNavigate: () => navigate }));
vi.mock('@inngest/components/ThemeMode/ModeSwitch', () => ({
  default: () => null,
}));
vi.mock('../Auth/SignOutButton', () => ({ SignOutButton: () => null }));

beforeEach(() => {
  navigate.mockClear();
  Element.prototype.scrollIntoView = vi.fn();
});
afterEach(cleanup);

it.each(['Enter', ' '])('opens OAuth sessions with %s', async (key) => {
  render(<ProfileMenu isMarketplace={false}>Profile</ProfileMenu>);
  fireEvent.click(screen.getByRole('button', { name: 'Profile' }));
  const menu = await screen.findByRole('listbox');
  for (let index = 0; index < 5; index++) {
    fireEvent.keyDown(menu, { key: 'ArrowDown' });
  }
  await waitFor(() =>
    expect(menu.getAttribute('aria-activedescendant')).toBe(
      screen.getByRole('option', { name: 'OAuth sessions' }).id,
    ),
  );
  fireEvent.keyDown(menu, { key });
  await waitFor(() =>
    expect(navigate).toHaveBeenCalledWith({ to: '/settings/oauth-sessions' }),
  );
});
