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
import { CombinedError } from 'urql';
import { SecretEditor } from './SecretEditor';

const mutations = vi.hoisted(() => ({ create: vi.fn(), replace: vi.fn() }));
vi.mock('urql', async (original) => ({
  ...(await original<typeof import('urql')>()),
  useMutation: (document: { definitions: { name?: { value: string } }[] }) => [
    {},
    document.definitions[0]?.name?.value === 'ReplaceSandboxSecret'
      ? mutations.replace
      : mutations.create,
  ],
}));
vi.mock('sonner', () => ({ toast: { success: vi.fn() } }));

const props = () => ({
  environmentID: 'environment-a',
  existingSecrets: [],
  onBack: vi.fn(),
  onSaved: vi.fn(),
  onDirtyChange: vi.fn(),
  onSavingChange: vi.fn(),
});
const paste = (text: string) =>
  fireEvent.paste(screen.getByLabelText('Secret 1 name'), {
    clipboardData: { getData: () => text },
  });

afterEach(cleanup);
beforeEach(() => {
  vi.resetAllMocks();
});

describe('secret editor', () => {
  it('pastes editable rows, masks values, and saves exact names and values in the environment', async () => {
    const callbacks = props();
    mutations.create.mockResolvedValue({
      data: { createEnvSecret: { id: 'saved' } },
    });
    render(<SecretEditor {...callbacks} />);
    paste('TOKEN="  value#1  "\nCERT="first\\nsecond"\nEMPTY=');
    expect(screen.getByLabelText('Secret 1 name').getAttribute('value')).toBe(
      'TOKEN',
    );
    expect(screen.getByLabelText('Secret 1 value').getAttribute('type')).toBe(
      'password',
    );
    fireEvent.click(
      screen.getByRole('button', { name: 'Show secret 2 value' }),
    );
    expect(
      screen.getByLabelText<HTMLTextAreaElement>('Secret 2 value').value,
    ).toBe('first\nsecond');
    fireEvent.click(screen.getByRole('button', { name: 'Save 3 Secrets' }));
    await waitFor(() => expect(callbacks.onSaved).toHaveBeenCalledOnce());
    expect(mutations.create.mock.calls.map(([input]) => input.input)).toEqual([
      { workspaceID: 'environment-a', name: 'TOKEN', value: '  value#1  ' },
      { workspaceID: 'environment-a', name: 'CERT', value: 'first\nsecond' },
      { workspaceID: 'environment-a', name: 'EMPTY', value: '' },
    ]);
    expect(screen.queryByLabelText('Secret 1 value')).toBeNull();
  });

  it('does not save duplicates or overwrite existing names', async () => {
    render(
      <SecretEditor
        {...props()}
        existingSecrets={[
          { id: 'old', name: 'EXISTING', createdAt: '', updatedAt: '' },
        ]}
      />,
    );
    paste('TOKEN=one\nTOKEN=two\nEXISTING=three');
    fireEvent.click(screen.getByRole('button', { name: 'Save 3 Secrets' }));
    expect(
      screen.getAllByText('This name appears more than once.'),
    ).toHaveLength(2);
    expect(screen.getByText(/Already saved/)).toBeTruthy();
    expect(mutations.create).not.toHaveBeenCalled();
    expect(mutations.replace).not.toHaveBeenCalled();
  });

  it('retains only unsaved rows after a partial failure and never resubmits the successful row', async () => {
    const callbacks = props();
    mutations.create
      .mockResolvedValueOnce({ data: { createEnvSecret: { id: 'saved' } } })
      .mockResolvedValueOnce({
        error: new CombinedError({
          networkError: new Error('private-provider-input'),
        }),
      });
    render(<SecretEditor {...callbacks} />);
    paste('FIRST=one\nSECOND=two\nTHIRD=three');
    fireEvent.click(screen.getByRole('button', { name: 'Save 3 Secrets' }));
    await screen.findByText(/1 saved. Could not confirm the next save/);
    expect(mutations.create).toHaveBeenCalledTimes(2);
    expect(screen.getByLabelText<HTMLInputElement>('Secret 1 name').value).toBe(
      'SECOND',
    );
    expect(screen.queryByText('private-provider-input')).toBeNull();
    mutations.create.mockResolvedValue({
      data: { createEnvSecret: { id: 'saved' } },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save 2 Secrets' }));
    await waitFor(() => expect(callbacks.onSaved).toHaveBeenCalledOnce());
    expect(
      mutations.create.mock.calls.map(([input]) => input.input.name),
    ).toEqual(['FIRST', 'SECOND', 'SECOND', 'THIRD']);
  });

  it('stops the remaining import when the environment changes/unmounts', async () => {
    let finish!: (value: unknown) => void;
    mutations.create.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const callbacks = props();
    const { unmount } = render(<SecretEditor {...callbacks} />);
    paste('FIRST=one\nSECOND=two');
    fireEvent.click(screen.getByRole('button', { name: 'Save 2 Secrets' }));
    expect(mutations.create).toHaveBeenCalledOnce();
    unmount();
    await act(async () => {
      finish({ data: { createEnvSecret: { id: 'saved' } } });
    });
    expect(mutations.create).toHaveBeenCalledOnce();
    expect(callbacks.onSaved).not.toHaveBeenCalled();
  });

  it('replaces by the saved UUID without renaming or exposing the old value', async () => {
    const callbacks = props();
    mutations.replace.mockResolvedValue({
      data: { updateEnvSecretValue: { id: 'original' } },
    });
    render(
      <SecretEditor
        {...callbacks}
        replacing={{
          id: 'original',
          name: 'TOKEN',
          createdAt: '',
          updatedAt: '',
        }}
      />,
    );
    expect(screen.queryByLabelText('Secret 1 name')).toBeNull();
    fireEvent.change(screen.getByLabelText('Secret 1 value'), {
      target: { value: 'new-value' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Replace Value' }));
    await waitFor(() => expect(callbacks.onSaved).toHaveBeenCalledOnce());
    expect(mutations.replace.mock.calls[0]?.[0]).toEqual({
      input: {
        workspaceID: 'environment-a',
        id: 'original',
        value: 'new-value',
      },
    });
    expect(mutations.create).not.toHaveBeenCalled();
  });
});
