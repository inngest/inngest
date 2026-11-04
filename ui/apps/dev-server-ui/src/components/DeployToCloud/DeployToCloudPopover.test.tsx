import type { ReactNode } from 'react';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

import { DeployToCloudPopover } from './DeployToCloudPopover';
import { DEPLOYMENT_PROMPT } from './prompt';

vi.mock('@inngest/components/Link', () => ({
  Link: ({
    children,
    to,
    href,
    onClick,
  }: {
    children: ReactNode;
    to?: string;
    href?: string;
    onClick?: () => void;
  }) => (
    <a
      href={href ?? to}
      onClick={(event) => {
        event.preventDefault();
        onClick?.();
      }}
    >
      {children}
    </a>
  ),
}));

beforeEach(() => {
  const modals = document.createElement('div');
  modals.id = 'modals';
  document.body.appendChild(modals);
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
  });
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
});

afterEach(() => {
  cleanup();
  document.getElementById('modals')?.remove();
  vi.unstubAllGlobals();
});

async function openPopover() {
  render(<DeployToCloudPopover />);
  fireEvent.click(screen.getByRole('button', { name: 'Deploy to Cloud' }));
  await screen.findByRole('dialog');
}

it('copies the reviewed prompt and only confirms the clipboard operation', async () => {
  await openPopover();
  expect(screen.queryByRole('textbox')).toBeNull();
  await act(async () => {
    fireEvent.click(
      screen.getByRole('button', { name: 'Copy deployment prompt' }),
    );
  });
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith(DEPLOYMENT_PROMPT);
  expect(screen.getByRole('button', { name: 'Copied!' })).toBeTruthy();
  expect(screen.queryByText(/successfully deployed/i)).toBeNull();
});

it('copies edits from the preview and disables copying an empty prompt', async () => {
  await openPopover();
  fireEvent.click(screen.getByRole('button', { name: 'View prompt' }));
  const textarea = screen.getByRole('textbox', {
    name: 'Cloud deployment prompt',
  });
  fireEvent.change(textarea, {
    target: { value: 'Deploy this specific app.' },
  });
  await act(async () => {
    fireEvent.click(
      screen.getByRole('button', { name: 'Copy deployment prompt' }),
    );
  });
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
    'Deploy this specific app.',
  );
  fireEvent.change(textarea, { target: { value: '  ' } });
  expect(
    screen.getByRole<HTMLButtonElement>('button', {
      name: 'Copy deployment prompt',
    }).disabled,
  ).toBe(true);
});

it('reveals and selects the prompt when clipboard access fails', async () => {
  vi.mocked(navigator.clipboard.writeText).mockRejectedValueOnce(
    new Error('denied'),
  );
  await openPopover();
  await act(async () => {
    fireEvent.click(
      screen.getByRole('button', { name: 'Copy deployment prompt' }),
    );
  });
  const textarea = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Cloud deployment prompt',
  });
  expect(screen.getByRole('status').textContent).toContain('Couldn’t copy');
  expect(document.activeElement).toBe(textarea);
  expect(textarea.selectionStart).toBe(0);
  expect(textarea.selectionEnd).toBe(DEPLOYMENT_PROMPT.length);
  expect(screen.queryByRole('button', { name: 'Copied!' })).toBeNull();
});

it('does not confirm an old copy after the prompt changes', async () => {
  let resolveCopy!: () => void;
  vi.mocked(navigator.clipboard.writeText).mockReturnValueOnce(
    new Promise<void>((resolve) => {
      resolveCopy = resolve;
    }),
  );
  await openPopover();
  fireEvent.click(screen.getByRole('button', { name: 'View prompt' }));
  fireEvent.click(
    screen.getByRole('button', { name: 'Copy deployment prompt' }),
  );
  fireEvent.change(screen.getByRole('textbox'), {
    target: { value: 'Use this updated deployment prompt.' },
  });
  await act(async () => {
    resolveCopy();
  });
  expect(screen.queryByRole('button', { name: 'Copied!' })).toBeNull();
  await act(async () => {
    fireEvent.click(
      screen.getByRole('button', { name: 'Copy deployment prompt' }),
    );
  });
  expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith(
    'Use this updated deployment prompt.',
  );
  expect(screen.getByRole('button', { name: 'Copied!' })).toBeTruthy();
});

it('shows the manual guide in place and returns to the edited prompt', async () => {
  await openPopover();
  fireEvent.click(screen.getByRole('button', { name: 'View prompt' }));
  fireEvent.change(screen.getByRole('textbox'), {
    target: { value: 'Deploy this app.' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Deploy manually' }));
  expect(screen.getByRole('dialog')).toBeTruthy();
  expect(screen.queryByRole('textbox')).toBeNull();
  expect(screen.getByRole('link', { name: 'Cloud' }).getAttribute('href')).toBe(
    'https://app.inngest.com',
  );
  expect(screen.getAllByRole('link')).toHaveLength(2);
  expect(
    screen.getByRole('link', { name: 'Full deployment guide' }),
  ).toBeTruthy();
  expect(document.activeElement).toBe(
    screen.getByRole('heading', { name: 'Deploy manually' }),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Back to AI setup' }));
  expect(screen.getByRole<HTMLTextAreaElement>('textbox').value).toBe(
    'Deploy this app.',
  );
  expect(document.activeElement).toBe(
    screen.getByRole('button', { name: 'Deploy manually' }),
  );
});
