import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { toast } from 'sonner';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { SendEventModal, type SendEventConfig } from './SendEventModal';

const editor = vi.hoisted(() => ({ run: undefined as undefined | (() => void) }));
vi.mock('@monaco-editor/react', () => ({
  default: ({
    value,
    onChange,
    onMount,
  }: {
    value?: string;
    onChange?: (value: string) => void;
    onMount?: (editor: { addAction: (action: { run: () => void }) => void }) => void;
  }) => {
    if (onMount) onMount({ addAction: ({ run }: { run: () => void }) => (editor.run = run) });
    return (
      <textarea
        aria-label="Event JSON"
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
    );
  },
  useMonaco: () => ({
    editor: { defineTheme: vi.fn() },
    languages: { json: { jsonDefaults: { setDiagnosticsOptions: vi.fn() } } },
    KeyMod: { CtrlCmd: 1 },
    KeyCode: { Enter: 2 },
  }),
}));
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const initialEvent = { name: 'app/original', data: { value: 1 } };
const generateCode = (value: unknown) => JSON.stringify(value);
const makeConfig = (sendEvent = vi.fn().mockResolvedValue(undefined)): SendEventConfig => ({
  sendEvent,
  generateSDKCode: generateCode,
  generateCurlCode: generateCode,
});
const shortcut = () => fireEvent.keyDown(document, { key: 'Enter', ctrlKey: true });
const changeJSON = (value: string) =>
  fireEvent.change(screen.getByLabelText('Event JSON'), { target: { value } });
const submit = () => fireEvent.click(screen.getByText('Send event').closest('button')!);

afterEach(() => {
  cleanup();
  document.getElementById('modals')?.remove();
});
beforeEach(() => {
  vi.clearAllMocks();
  editor.run = undefined;
  const modals = document.createElement('div');
  modals.id = 'modals';
  document.body.appendChild(modals);
});

describe('SendEventModal submission', () => {
  it.each(['{"name":', '', '   '])(
    'does not send stale data for invalid editor text %j',
    async (input) => {
      const config = makeConfig();
      render(
        <SendEventModal
          isOpen
          data={JSON.stringify(initialEvent)}
          onClose={vi.fn()}
          config={config}
        />
      );
      changeJSON(input);
      submit();
      await act(async () => {});
      expect(config.sendEvent).not.toHaveBeenCalled();
    }
  );

  it.each(['null', '[null]', '[{"name":"app/ok"},null]'])(
    'rejects null event entries in %s without throwing',
    async (input) => {
      const config = makeConfig();
      render(
        <SendEventModal
          isOpen
          data={JSON.stringify(initialEvent)}
          onClose={vi.fn()}
          config={config}
        />
      );
      changeJSON(input);
      await act(async () => {
        await editor.run?.();
      });
      expect(config.sendEvent).not.toHaveBeenCalled();
      expect(toast.error).toHaveBeenCalledWith('Each event payload must be an object.');
    }
  );

  it('sends the current valid event as a batch', async () => {
    const config = makeConfig();
    const close = vi.fn();
    render(
      <SendEventModal isOpen data={JSON.stringify(initialEvent)} onClose={close} config={config} />
    );
    const changed = { name: 'app/changed', data: { value: 2 } };
    changeJSON(JSON.stringify(changed));
    submit();
    await waitFor(() => expect(close).toHaveBeenCalledOnce());
    expect(config.sendEvent).toHaveBeenCalledExactlyOnceWith([changed]);
  });

  it('sends a valid batch without changing its entries', async () => {
    const config = makeConfig();
    render(
      <SendEventModal
        isOpen
        data={JSON.stringify(initialEvent)}
        onClose={vi.fn()}
        config={config}
      />
    );
    const batch = [initialEvent, { name: 'app/second', user: { id: 'synthetic' } }];
    changeJSON(JSON.stringify(batch));
    shortcut();
    await waitFor(() => expect(config.sendEvent).toHaveBeenCalledExactlyOnceWith(batch));
  });

  it('preserves the existing empty batch behavior', async () => {
    const config = makeConfig();
    render(
      <SendEventModal
        isOpen
        data={JSON.stringify(initialEvent)}
        onClose={vi.fn()}
        config={config}
      />
    );
    changeJSON('[]');
    submit();
    await waitFor(() => expect(config.sendEvent).toHaveBeenCalledExactlyOnceWith([]));
  });

  it('does not dispatch invalid text through the keyboard shortcut', async () => {
    const config = makeConfig();
    render(
      <SendEventModal
        isOpen
        data={JSON.stringify(initialEvent)}
        onClose={vi.fn()}
        config={config}
      />
    );
    changeJSON('{');
    shortcut();
    await act(async () => {});
    expect(config.sendEvent).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith('Event payload must be valid JSON.');
  });

  it('rejects the whole batch when one event has no name', async () => {
    const config = makeConfig();
    render(
      <SendEventModal
        isOpen
        data={JSON.stringify(initialEvent)}
        onClose={vi.fn()}
        config={config}
      />
    );
    changeJSON(JSON.stringify([initialEvent, { data: {} }]));
    submit();
    await act(async () => {});
    expect(config.sendEvent).not.toHaveBeenCalled();
  });

  it('ignores keyboard submit while the external request is loading', async () => {
    const config = { ...makeConfig(), ui: { isLoading: true } };
    render(
      <SendEventModal
        isOpen
        data={JSON.stringify(initialEvent)}
        onClose={vi.fn()}
        config={config}
      />
    );
    shortcut();
    await act(async () => {});
    expect(config.sendEvent).not.toHaveBeenCalled();
  });

  it('allows only one in-flight request across button and both shortcut paths', async () => {
    let finish!: () => void;
    const sendEvent = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        })
    );
    const config = makeConfig(sendEvent);
    render(
      <SendEventModal
        isOpen
        data={JSON.stringify(initialEvent)}
        onClose={vi.fn()}
        config={config}
      />
    );
    act(() => {
      submit();
      shortcut();
      editor.run?.();
      shortcut();
    });
    expect(sendEvent).toHaveBeenCalledOnce();
    await act(async () => finish());
  });

  it('ignores shortcuts after a mounted modal closes', async () => {
    const config = makeConfig();
    const props = { data: JSON.stringify(initialEvent), onClose: vi.fn(), config };
    const { rerender } = render(<SendEventModal {...props} isOpen />);
    rerender(<SendEventModal {...props} isOpen={false} />);
    shortcut();
    await act(async () => {});
    expect(config.sendEvent).not.toHaveBeenCalled();
  });

  it('allows retry after a failed request', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    const sendEvent = vi
      .fn()
      .mockRejectedValueOnce(new Error('synthetic failure'))
      .mockResolvedValue(undefined);
    const config = makeConfig(sendEvent);
    const close = vi.fn();
    render(
      <SendEventModal isOpen data={JSON.stringify(initialEvent)} onClose={close} config={config} />
    );
    submit();
    await waitFor(() => expect(error).toHaveBeenCalled());
    shortcut();
    await waitFor(() => expect(close).toHaveBeenCalledOnce());
    expect(sendEvent).toHaveBeenCalledTimes(2);
    error.mockRestore();
  });
});
