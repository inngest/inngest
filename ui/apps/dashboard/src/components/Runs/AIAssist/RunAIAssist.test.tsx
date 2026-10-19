// @vitest-environment jsdom
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RunAIAssistPopover } from './RunAIAssistPopover';
import type { RunPromptContext } from './prompt';

const context: RunPromptContext = {
  account: 'Example',
  environmentSlug: 'production',
  scope: 'env',
  search: {},
};
function mount(overrides = {}) {
  const onAction = vi.fn();
  const props = {
    context,
    metadataLoading: false,
    metadataError: false,
    onRetry: vi.fn(),
    onAction,
    getSetupHref: (client: string) => `/mcp/setup?client=${client}`,
    ...overrides,
  };
  return { ...render(<RunAIAssistPopover {...props} />), props, onAction };
}
function open() {
  fireEvent.click(screen.getByRole('button', { name: 'Find runs with AI' }));
}

beforeEach(() => {
  const modals = document.createElement('div');
  modals.id = 'modals';
  document.body.appendChild(modals);
});
afterEach(() => {
  cleanup();
  document.getElementById('modals')?.remove();
  vi.restoreAllMocks();
});

describe('Runs AI popover', () => {
  it('tracks one view across rerenders and each intentional opening', async () => {
    const { rerender, props, onAction } = mount();
    rerender(<RunAIAssistPopover {...props} />);
    open();
    fireEvent.keyDown(document, { key: 'Escape' });
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    open();
    expect(
      onAction.mock.calls.filter(([event]) => event.action === 'viewed'),
    ).toHaveLength(1);
    expect(
      onAction.mock.calls.filter(([event]) => event.action === 'opened'),
    ).toHaveLength(2);
  });

  it('copies the current prompt and tracks success only after the clipboard resolves', async () => {
    let resolve!: () => void;
    const writeText = vi.fn(
      () =>
        new Promise<void>((done) => {
          resolve = done;
        }),
    );
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    });
    const { onAction } = mount();
    open();
    fireEvent.click(screen.getByRole('button', { name: 'Copy prompt' }));
    expect(onAction).toHaveBeenCalledWith({ action: 'prompt_copy_clicked' });
    expect(onAction).not.toHaveBeenCalledWith({ action: 'prompt_copied' });
    expect(writeText).toHaveBeenCalledWith(
      screen.getByLabelText('Prompt preview').textContent,
    );
    await act(async () => resolve());
    expect(onAction).toHaveBeenCalledWith({ action: 'prompt_copied' });
    expect(screen.getByRole('button', { name: 'Copied' })).toBeTruthy();
  });

  it('keeps selectable text after clipboard failure and records its category', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: vi
          .fn()
          .mockRejectedValue(new DOMException('denied', 'NotAllowedError')),
      },
      configurable: true,
    });
    const { onAction } = mount();
    open();
    fireEvent.click(screen.getByRole('button', { name: 'Copy prompt' }));
    await screen.findByText('Couldn’t copy. Select and copy the prompt above.');
    expect(screen.getByLabelText('Prompt preview')).toBeTruthy();
    expect(onAction).toHaveBeenCalledWith({
      action: 'prompt_copy_failed',
      failure_category: 'permission_denied',
    });
    expect(onAction).not.toHaveBeenCalledWith({ action: 'prompt_copied' });
  });

  it('blocks advanced filters while leaving setup links available and tracks client selection', () => {
    const { onAction } = mount({
      context: { ...context, search: { search: 'private CEL' } },
    });
    open();
    const copy = screen.getByRole('button', {
      name: 'Copy prompt',
    });
    expect(copy.hasAttribute('disabled')).toBe(true);
    expect(screen.getByText(/Clear CEL search/)).toBeTruthy();
    const link = screen.getByRole('link', { name: 'Codex CLI' });
    link.addEventListener('click', (event) => event.preventDefault());
    fireEvent.click(link);
    expect(link.getAttribute('href')).toBe('/mcp/setup?client=codex');
    expect(onAction).toHaveBeenCalledWith({
      action: 'setup_clicked',
      client: 'codex_cli',
    });
    expect(onAction).not.toHaveBeenCalledWith({
      action: 'prompt_copy_clicked',
    });
  });

  it('does not show stale copy success after filters change during a pending copy', async () => {
    let resolve!: () => void;
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: () =>
          new Promise<void>((done) => {
            resolve = done;
          }),
      },
      configurable: true,
    });
    const { props, rerender } = mount();
    open();
    fireEvent.click(screen.getByRole('button', { name: 'Copy prompt' }));
    rerender(
      <RunAIAssistPopover
        {...props}
        context={{ ...context, search: { filterStatus: '["FAILED"]' } }}
      />,
    );
    await act(async () => resolve());
    expect(screen.queryByRole('button', { name: 'Copied' })).toBeNull();
    expect(screen.getByLabelText('Prompt preview').textContent).toContain(
      'FAILED',
    );
  });

  it('offers metadata retry and keeps copying disabled while loading', () => {
    const { props, rerender } = mount({ metadataLoading: true });
    open();
    expect(
      screen
        .getByRole('button', { name: 'Copy prompt' })
        .hasAttribute('disabled'),
    ).toBe(true);
    rerender(
      <RunAIAssistPopover {...props} metadataLoading={false} metadataError />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(props.onRetry).toHaveBeenCalledOnce();
  });
});
