import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ManualSetup } from './ManualSetup';

const trackEvent = vi.hoisted(() => vi.fn());
vi.mock('@/hooks/useTracking', () => ({ useTracking: () => ({ trackEvent }) }));
vi.mock('./QuickstartProvider', () => ({
  useQuickstart: () => ({ server: 'http://localhost:9999' }),
}));
// Exercise the real command tabs and copy control without starting Monaco.
vi.mock('@monaco-editor/react', () => ({
  useMonaco: () => null,
  default: () => null,
}));
vi.mock('@/store/devApi', () => ({
  useInfoQuery: () => ({ data: { isSingleNodeService: false } }),
}));
vi.mock('@/store/generated', () => ({
  useCreateAppMutation: () => [vi.fn()],
}));

beforeEach(() => {
  trackEvent.mockClear();
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
  });
});
afterEach(cleanup);

it.each([
  {
    discoveryEnabled: true,
    disconnected: false,
    message: 'Auto-detecting apps…',
  },
  {
    discoveryEnabled: false,
    disconnected: false,
    message: 'Automatic discovery is disabled. Sync your app URL manually.',
  },
  {
    discoveryEnabled: true,
    disconnected: true,
    message: 'Your app is disconnected. Start it again or sync its URL.',
  },
])(
  'offers manual sync: $message',
  ({ discoveryEnabled, disconnected, message }) => {
    render(
      <ManualSetup
        discoveryEnabled={discoveryEnabled}
        disconnected={disconnected}
      />,
    );
    expect(screen.getByRole('status').textContent).toBe(message);
    expect(screen.getByRole('button', { name: 'Sync manually' })).toBeTruthy();
    expect(screen.queryByRole('link', { name: 'View apps' })).toBeNull();
    const errors = screen.queryByRole('link', {
      name: 'View app connection errors',
    });
    expect(errors?.getAttribute('href') ?? null).toBe(
      disconnected ? '/apps' : null,
    );
  },
);

it.each([
  { connectedAppNames: ['Tiny Notes'], message: 'Tiny Notes is connected.' },
  {
    connectedAppNames: ['Tiny Notes', 'Orders'],
    message: 'Tiny Notes and 1 more app are connected.',
  },
  {
    connectedAppNames: ['Tiny Notes', 'Orders', 'Reports'],
    message: 'Tiny Notes and 2 more apps are connected.',
  },
])(
  'shows connected app names and links to Apps: $message',
  ({ connectedAppNames, message }) => {
    render(
      <ManualSetup
        discoveryEnabled={false}
        disconnected={false}
        connectedAppNames={connectedAppNames}
      />,
    );
    expect(screen.getByRole('status').textContent).toBe(message);
    expect(
      screen.getByRole('link', { name: 'View apps' }).getAttribute('href'),
    ).toBe('/apps');
    expect(screen.queryByRole('button', { name: 'Sync manually' })).toBeNull();
    expect(
      screen.queryByRole('link', { name: 'View app connection errors' }),
    ).toBeNull();
  },
);

async function selectLanguage(name: string) {
  const button = screen.getByRole('button', { name: /^Language / });
  button.focus();
  fireEvent.keyDown(button, { key: 'ArrowDown' });
  fireEvent.click(await screen.findByRole('option', { name }));
}

function selectManager(name: string) {
  fireEvent.mouseDown(screen.getByRole('tab', { name }), {
    button: 0,
    ctrlKey: false,
  });
}

async function expectCommand(command: string) {
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: 'Copy' })),
  );
  expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith(command);
}

it('switches language-specific tabs, commands and guides without keeping an incompatible manager', async () => {
  render(<ManualSetup discoveryEnabled disconnected={false} />);
  expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
    'npm',
    'yarn',
    'pnpm',
    'bun',
  ]);
  selectManager('pnpm');
  await expectCommand('pnpm add inngest');

  await selectLanguage('Python');
  expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
    'pip',
    'uv',
  ]);
  expect(
    screen.getByRole('tab', { name: 'pip' }).getAttribute('aria-selected'),
  ).toBe('true');
  expect(
    screen
      .getByRole('link', { name: 'Open Python setup guide' })
      .getAttribute('href'),
  ).toContain('guide=python');
  await expectCommand('pip install inngest');
  selectManager('uv');
  await expectCommand('uv add inngest');

  await selectLanguage('Go');
  expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
    'go',
  ]);
  expect(
    screen
      .getByRole('link', { name: 'Open Go setup guide' })
      .getAttribute('href'),
  ).toContain('guide=go');
  await expectCommand('go get github.com/inngest/inngestgo');

  await selectLanguage('TypeScript');
  expect(
    screen.getByRole('tab', { name: 'npm' }).getAttribute('aria-selected'),
  ).toBe('true');
  expect(
    screen
      .getByRole('link', { name: 'Open TypeScript setup guide' })
      .getAttribute('href'),
  ).toContain('guide=typescript');
  await expectCommand('npm install inngest');
});

it('tracks the selected language and manager on copy and the selected language on guide clicks', async () => {
  render(<ManualSetup discoveryEnabled disconnected={false} />);
  expect(trackEvent).not.toHaveBeenCalled();
  await selectLanguage('Python');
  selectManager('uv');
  await expectCommand('uv add inngest');
  const guide = screen.getByRole('link', { name: 'Open Python setup guide' });
  guide.addEventListener('click', (event) => event.preventDefault(), {
    once: true,
  });
  fireEvent.click(guide);
  expect(trackEvent.mock.calls).toEqual([
    [
      'cli/onboarding.action',
      {
        flow: 'quickstart',
        version: 1,
        action: 'manual_install_copy_clicked',
        language: 'Python',
        package_manager: 'uv',
      },
    ],
    [
      'cli/onboarding.action',
      {
        flow: 'quickstart',
        version: 1,
        action: 'docs_clicked',
        guide: 'manual',
        language: 'Python',
      },
    ],
  ]);
});
