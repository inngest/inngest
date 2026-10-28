import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { FirstRun } from './FirstRun';
import { AgentSetup } from './AgentSetup';
import { Quickstart } from './Quickstart';

const fixtures = vi.hoisted(() => ({
  context: {} as any,
  details: {} as any,
  invoke: vi.fn(),
  setAttempt: vi.fn(),
  track: vi.fn(),
  refetchFunctions: vi.fn(),
  navigate: vi.fn(),
}));
vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  Navigate: (props: unknown) => {
    fixtures.navigate(props);
    return null;
  },
}));
vi.mock('./ManualSetup', () => ({
  ManualSetup: () => <div>Manual setup guide</div>,
}));
vi.mock('@/hooks/useTracking', () => ({
  useTracking: () => ({ trackEvent: fixtures.track }),
}));
vi.mock('./QuickstartProvider', () => ({
  useQuickstart: () => fixtures.context,
}));
vi.mock('@/store/generated', () => ({
  FunctionTriggerTypes: { Event: 'EVENT' },
  useGetFunctionsQuery: () => fixtures.details,
  useInvokeFunctionMutation: () => [fixtures.invoke, { isLoading: false }],
}));
vi.mock('@inngest/components/CodeBlock', () => {
  const CodeBlock = ({ tab }: any) => (
    <textarea
      aria-label="Payload"
      value={tab.content}
      onChange={(event) => tab.handleChange(event.target.value)}
    />
  );
  CodeBlock.Wrapper = ({ children }: any) => children;
  return { CodeBlock };
});
const fn = { id: 'fn-1', slug: 'app-hello', name: 'Hello', appName: 'App' };
const invokeButton = () =>
  screen.getByRole<HTMLButtonElement>('button', { name: 'Invoke function' });
const copyPrompt = () =>
  act(async () =>
    fireEvent.click(
      screen.getByRole('button', { name: 'Copy onboarding prompt' }),
    ),
  );

beforeEach(() => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
  });
  fixtures.invoke.mockReset();
  fixtures.navigate.mockClear();
  fixtures.track.mockClear();
  fixtures.refetchFunctions.mockClear();
  fixtures.setAttempt.mockReset();
  fixtures.context = {
    server: 'http://localhost:9999',
    functions: [fn],
    attempt: null,
    setAttempt: fixtures.setAttempt,
    candidates: [],
    runs: {},
    run: null,
    completion: null,
  };
  fixtures.details = {
    refetch: fixtures.refetchFunctions,
    data: { functions: [{ ...fn, triggers: [{ type: 'EVENT' }] }] },
  };
});
afterEach(cleanup);
it('validates the existing payload format before invoking', () => {
  render(<FirstRun />);
  fireEvent.change(screen.getByLabelText('Payload'), {
    target: { value: '{"data": []}' },
  });
  fireEvent.click(invokeButton());
  expect(fixtures.invoke).not.toHaveBeenCalled();
  expect(fixtures.track).not.toHaveBeenCalled();
  expect(
    screen.getByText('The "data" field must be an object or null'),
  ).toBeTruthy();
});
it('invokes once with the existing mutation and keeps acceptance distinct from completion', async () => {
  let finish: (value: unknown) => void;
  fixtures.invoke.mockReturnValue({
    unwrap: () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  });
  const view = render(<FirstRun />);
  fireEvent.change(screen.getByLabelText('Payload'), {
    target: { value: '{"data":{"name":"Ada"}}' },
  });
  fireEvent.click(invokeButton());
  fireEvent.click(invokeButton());
  expect(fixtures.invoke).toHaveBeenCalledExactlyOnceWith({
    functionSlug: fn.slug,
    data: { name: 'Ada' },
    user: null,
    meta: null,
  });
  expect(fixtures.track).toHaveBeenCalledExactlyOnceWith(
    'cli/onboarding.action',
    {
      flow: 'quickstart',
      version: 1,
      action: 'invoke_clicked',
      path: 'manual',
    },
  );
  fixtures.context = {
    ...fixtures.context,
    attempt: fixtures.setAttempt.mock.calls[0][0],
  };
  await act(async () => {
    finish!({ invokeFunction: true });
  });
  view.rerender(<FirstRun />);
  expect(invokeButton().disabled).toBe(true);
  expect(fixtures.navigate).not.toHaveBeenCalled();
});
it('retains the payload on a rejected invocation so it can be corrected and retried', async () => {
  fixtures.invoke.mockReturnValue({
    unwrap: async () => ({ invokeFunction: false }),
  });
  render(<FirstRun />);
  const payload = '{"data":{"name":"Ada"}}';
  fireEvent.change(screen.getByLabelText('Payload'), {
    target: { value: payload },
  });
  await act(async () => fireEvent.click(invokeButton()));
  expect(screen.getByLabelText<HTMLTextAreaElement>('Payload').value).toBe(
    payload,
  );
  expect(fixtures.setAttempt).toHaveBeenLastCalledWith(null);
  expect(
    screen.getByText(
      'The invocation was not accepted. Check your app connection and try again.',
    ),
  ).toBeTruthy();
});
it('invokes cron-only functions without payload data', async () => {
  fixtures.details.data.functions[0].triggers = [{ type: 'CRON' }];
  fixtures.invoke.mockReturnValue({
    unwrap: async () => ({ invokeFunction: true }),
  });
  render(<FirstRun />);
  expect(screen.queryByLabelText('Payload')).toBeNull();
  await act(async () => fireEvent.click(invokeButton()));
  expect(fixtures.invoke).toHaveBeenCalledWith({
    functionSlug: fn.slug,
    data: {},
    user: null,
    meta: null,
  });
});

it('starts agent observation on prompt handoff without claiming a connection or invoking', async () => {
  render(<AgentSetup server="http://localhost:9999" />);
  expect(screen.queryByLabelText('Onboarding prompt')).toBeNull();
  expect(
    screen.getByRole('link', { name: 'MCP guide' }).getAttribute('href'),
  ).toBe(
    'https://www.inngest.com/docs/ai-dev-tools/mcp#connect-to-the-dev-server',
  );
  await copyPrompt();
  expect(fixtures.setAttempt).toHaveBeenCalledWith(
    expect.objectContaining({
      mode: 'agent',
      functionID: '',
      startedAt: expect.any(String),
    }),
  );
  expect(fixtures.invoke).not.toHaveBeenCalled();
  expect(fixtures.track).toHaveBeenCalledWith('cli/onboarding.action', {
    flow: 'quickstart',
    version: 1,
    action: 'agent_prompt_copied',
  });
});

it('keeps the prompt hidden until requested and copies reviewed edits', async () => {
  render(<AgentSetup server="http://localhost:9999" />);
  fireEvent.click(screen.getByRole('button', { name: 'View prompt' }));
  expect(
    screen.getByLabelText<HTMLTextAreaElement>('Onboarding prompt').value,
  ).toContain('http://localhost:9999/mcp');
  fireEvent.change(screen.getByLabelText('Onboarding prompt'), {
    target: { value: 'My reviewed prompt' },
  });
  await copyPrompt();
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
    'My reviewed prompt',
  );
  fireEvent.click(screen.getByRole('button', { name: 'Hide prompt' }));
  expect(screen.queryByLabelText('Onboarding prompt')).toBeNull();
});
it('reveals a selectable prompt on clipboard failure without starting observation', async () => {
  vi.mocked(navigator.clipboard.writeText).mockRejectedValueOnce(
    new Error('Clipboard unavailable'),
  );
  render(<AgentSetup server="http://localhost:9999" />);
  await copyPrompt();
  expect(screen.getByLabelText('Onboarding prompt')).toBeTruthy();
  expect(
    screen.getByText('Couldn’t copy. Select and copy the prompt above.'),
  ).toBeTruthy();
  expect(fixtures.setAttempt).not.toHaveBeenCalled();
  expect(fixtures.track).not.toHaveBeenCalled();
});

it('keeps agent observation silent without adding a waiting message', () => {
  fixtures.context.attempt = {
    mode: 'agent',
    functionID: '',
    startedAt: '2026-10-23T12:00:00Z',
  };
  render(<AgentSetup server="http://localhost:9999" />);
  expect(screen.queryByText(/Waiting for your agent/)).toBeNull();
  expect(
    screen.getByRole('button', { name: 'Copy onboarding prompt' }),
  ).toBeTruthy();
  expect(screen.getByRole('button', { name: 'View prompt' })).toBeTruthy();
});

it('retries failed function details without invoking or losing the payload', () => {
  const view = render(<FirstRun />);
  fireEvent.change(screen.getByLabelText('Payload'), {
    target: { value: '{"data":{"name":"Ada"}}' },
  });
  fixtures.details = { ...fixtures.details, error: { message: 'offline' } };
  view.rerender(<FirstRun />);
  expect(invokeButton().disabled).toBe(true);
  fireEvent.click(
    screen.getByRole('button', { name: 'Retry function details' }),
  );
  expect(fixtures.refetchFunctions).toHaveBeenCalledTimes(1);
  expect(fixtures.invoke).not.toHaveBeenCalled();
  delete fixtures.details.error;
  view.rerender(<FirstRun />);
  expect(screen.getByLabelText<HTMLTextAreaElement>('Payload').value).toBe(
    '{"data":{"name":"Ada"}}',
  );
  expect(invokeButton().disabled).toBe(false);
});

it('shows a disabled function selector until a synced function becomes available', () => {
  fixtures.context.functions = [];
  const view = render(<FirstRun />);
  const selector = screen.getByRole('button', { name: /^Function / });
  expect(selector.hasAttribute('disabled')).toBe(true);
  expect(screen.getByText('Select a function to invoke')).toBeTruthy();
  expect(
    screen.getByText(
      'Your first run will appear here once the function is invoked',
    ),
  ).toBeTruthy();
  fireEvent.click(selector);
  expect(screen.queryByRole('listbox')).toBeNull();
  expect(screen.queryByRole('button', { name: 'Invoke function' })).toBeNull();
  fixtures.context.functions = [fn];
  view.rerender(<FirstRun />);
  expect(
    screen.getByRole('button', { name: /^Function / }).hasAttribute('disabled'),
  ).toBe(false);
  expect(screen.getByText('App / Hello')).toBeTruthy();
  expect(screen.getByLabelText('Payload')).toBeTruthy();
  expect(fixtures.invoke).not.toHaveBeenCalled();
  expect(fixtures.setAttempt).not.toHaveBeenCalled();
});

it.each([
  {
    source: 'saved completion',
    context: { completion: { runID: 'identified-run' } },
  },
  {
    source: 'explicit attempt',
    context: {
      attempt: {
        functionID: fn.id,
        startedAt: '2026-10-23T12:00:00Z',
        runID: 'identified-run',
      },
    },
  },
  ...['COMPLETED', 'QUEUED', 'RUNNING', 'FAILED', 'CANCELLED', 'SKIPPED'].map(
    (status) => ({
      source: `${status} run`,
      context: { run: { id: 'identified-run', status } },
    }),
  ),
])('opens the exact run from $source in Runs', ({ context }) => {
  Object.assign(fixtures.context, context);
  render(<FirstRun />);
  expect(fixtures.navigate).toHaveBeenCalledWith({
    to: '/runs',
    search: { runID: 'identified-run' },
    replace: true,
  });
});
it('uses only button loading while an accepted invocation has no identified run', () => {
  fixtures.context.attempt = {
    functionID: fn.id,
    startedAt: '2026-10-23T12:00:00Z',
  };
  render(<FirstRun />);
  expect(fixtures.navigate).not.toHaveBeenCalled();
  expect(invokeButton().disabled).toBe(true);
  expect(screen.queryByText(/Waiting for a run/)).toBeNull();
  expect(screen.queryByText(/Run status:/)).toBeNull();
});

it.each(['saved', 'historical'])(
  'reopens the %s completed setup guide without redirecting or clearing progress',
  (source) => {
    if (source === 'saved')
      fixtures.context.completion = { runID: 'completed-run' };
    else fixtures.context.run = { id: 'completed-run', status: 'COMPLETED' };
    fixtures.context.apps = { data: { apps: [{ connected: true }] } };
    fixtures.context.info = { data: { isDiscoveryEnabled: false } };
    render(<Quickstart showSetupGuide />);
    expect(
      screen.getByRole('heading', { name: 'Set up with a coding agent' }),
    ).toBeTruthy();
    expect(screen.getByText('Manual setup guide')).toBeTruthy();
    expect(
      screen.getByRole('link', { name: 'View your run' }).getAttribute('href'),
    ).toBe('/runs?runID=completed-run');
    expect(fixtures.navigate).not.toHaveBeenCalled();
    expect(fixtures.setAttempt).not.toHaveBeenCalled();
    expect(fixtures.invoke).not.toHaveBeenCalled();
  },
);

it('keeps step 3 invokable when reopening quickstart after completion and opens only the new run', async () => {
  fixtures.context.completion = { runID: 'previous-run' };
  fixtures.context.run = { id: 'previous-run', status: 'COMPLETED' };
  fixtures.invoke.mockReturnValue({
    unwrap: async () => ({ invokeFunction: true }),
  });
  const view = render(<FirstRun showSetupGuide />);
  expect(
    screen.getByRole('heading', { name: '3. Invoke your first function' }),
  ).toBeTruthy();
  expect(invokeButton().disabled).toBe(false);
  expect(fixtures.navigate).not.toHaveBeenCalled();
  await act(async () => fireEvent.click(invokeButton()));
  fixtures.context = {
    ...fixtures.context,
    attempt: fixtures.setAttempt.mock.calls[0][0],
    run: null,
  };
  view.rerender(<FirstRun showSetupGuide />);
  expect(fixtures.navigate).not.toHaveBeenCalled();
  fixtures.context.run = { id: 'new-run', status: 'RUNNING' };
  view.rerender(<FirstRun showSetupGuide />);
  expect(fixtures.navigate).toHaveBeenCalledWith({
    to: '/runs',
    search: { runID: 'new-run' },
    replace: true,
  });
});

it.each(['', fn.id])(
  'allows switching agent observation for function %j to manual invocation without an error',
  async (functionID) => {
    fixtures.context.attempt = {
      mode: 'agent',
      functionID,
      startedAt: '2026-10-23T12:00:00Z',
    };
    fixtures.invoke.mockReturnValue({
      unwrap: async () => ({ invokeFunction: true }),
    });
    const view = render(<FirstRun />);
    expect(invokeButton().disabled).toBe(!!functionID);
    expect(
      screen.getByText(
        'Pause your agent before switching to avoid duplicate runs.',
      ),
    ).toBeTruthy();
    fireEvent.click(
      screen.getByRole('button', { name: 'Switch to manual invocation' }),
    );
    expect(fixtures.setAttempt).toHaveBeenLastCalledWith(null);
    expect(fixtures.invoke).not.toHaveBeenCalled();
    fixtures.context.attempt = null;
    view.rerender(<FirstRun />);
    expect(invokeButton().disabled).toBe(false);
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: /^Function / })
        .disabled,
    ).toBe(false);
    expect(
      screen.queryByRole('button', { name: 'Switch to manual invocation' }),
    ).toBeNull();
    await act(async () => fireEvent.click(invokeButton()));
    expect(fixtures.invoke).toHaveBeenCalledTimes(1);
  },
);

it('links to the full runs list after an uncertain invocation without automatically resending', async () => {
  fixtures.invoke.mockReturnValue({
    unwrap: async () => {
      throw new Error('Lost response');
    },
  });
  const view = render(<FirstRun />);
  await act(async () => fireEvent.click(invokeButton()));
  fixtures.context.attempt = fixtures.setAttempt.mock.calls[0][0];
  view.rerender(<FirstRun />);
  expect(
    screen.getByRole('link', { name: 'Check Runs' }).getAttribute('href'),
  ).toBe('/runs?showAllRuns=true');
  expect(fixtures.setAttempt).not.toHaveBeenCalledWith(null);
  fireEvent.click(invokeButton());
  expect(fixtures.invoke).toHaveBeenCalledTimes(1);
});
