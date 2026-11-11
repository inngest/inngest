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

const sendTelemetry = vi.fn();

function trackedActions() {
  return sendTelemetry.mock.calls.map(([, request]) =>
    JSON.parse(request.body),
  );
}

function trackingEvent(action: string) {
  return { eventName: 'cli/dev_ui.deploy_to_cloud.action', action };
}

beforeEach(() => {
  sendTelemetry.mockReset().mockResolvedValue({ ok: true });
  vi.stubGlobal('fetch', sendTelemetry);
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
  const preview = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Cloud deployment prompt',
  });
  expect(preview.value).toBe(DEPLOYMENT_PROMPT);
  expect(preview.readOnly).toBe(true);
  await act(async () => {
    fireEvent.click(
      screen.getByRole('button', { name: 'Copy deployment prompt' }),
    );
  });
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith(DEPLOYMENT_PROMPT);
  expect(screen.getByRole('button', { name: 'Copied!' })).toBeTruthy();
  expect(screen.queryByText(/successfully deployed/i)).toBeNull();
  expect(trackedActions()).toEqual([
    trackingEvent('opened'),
    trackingEvent('prompt_copied'),
  ]);
});

it('selects the prompt when clipboard access fails', async () => {
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
  expect(trackedActions()).toEqual([trackingEvent('opened')]);
  expect(screen.queryByRole('button', { name: 'Copied!' })).toBeNull();
});

it('shows the manual guide in place and returns to the prompt preview', async () => {
  await openPopover();
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
  fireEvent.click(screen.getByRole('link', { name: 'Cloud' }));
  fireEvent.click(screen.getByRole('link', { name: 'Full deployment guide' }));
  expect(trackedActions()).toEqual(
    ['opened', 'manual_opened', 'cloud_clicked', 'docs_clicked'].map(
      trackingEvent,
    ),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Back to AI setup' }));
  expect(
    screen.getByRole('textbox', { name: 'Cloud deployment prompt' }),
  ).toBeTruthy();
  expect(document.activeElement).toBe(
    screen.getByRole('button', { name: 'Deploy manually' }),
  );
});

it('keeps the hidden panel out of the tab order in both views', async () => {
  await openPopover();
  const prompt = screen.getByRole('textbox', {
    name: 'Cloud deployment prompt',
  });
  const cloudLink = screen.getByText('Cloud');
  const manualButton = screen.getByRole('button', { name: 'Deploy manually' });

  expect(prompt.closest('[inert]')).toBeNull();
  expect(cloudLink.closest('[inert]')).not.toBeNull();

  fireEvent.click(manualButton);

  expect(prompt.closest('[inert]')).not.toBeNull();
  expect(manualButton.closest('[inert]')).not.toBeNull();
  expect(cloudLink.closest('[inert]')).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'Back to AI setup' }));

  expect(prompt.closest('[inert]')).toBeNull();
  expect(cloudLink.closest('[inert]')).not.toBeNull();
  expect(document.activeElement).toBe(manualButton);
});
