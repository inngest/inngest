import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './DropdownMenu';

afterEach(cleanup);

it('activates an action once and never activates disabled commands', () => {
  const enabled = vi.fn(),
    disabled = vi.fn();
  render(
    <DropdownMenu open>
      <DropdownMenuTrigger>Actions</DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem disabled onSelect={disabled}>
          Unavailable
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={enabled}>Run action</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
  fireEvent.click(screen.getByRole('menuitem', { name: 'Unavailable' }));
  expect(disabled).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('menuitem', { name: 'Run action' }));
  expect(enabled).toHaveBeenCalledTimes(1);
});
