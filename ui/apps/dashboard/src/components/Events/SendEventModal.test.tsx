import { renderToString } from 'react-dom/server';
import { print, type DocumentNode } from 'graphql';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SendEventConfig } from '@inngest/components/SendEvent/SendEventModal';

import { SendEventModal } from './SendEventModal';

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
  post: vi.fn(),
  config: undefined as SendEventConfig | undefined,
}));

vi.mock('urql', () => ({ useQuery: mocks.useQuery }));
vi.mock('ky', () => ({ default: { post: mocks.post } }));
vi.mock('@tanstack/react-router', () => ({ useRouter: () => ({}) }));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ id: 'environment-id', name: 'Production' }),
}));
vi.mock('@inngest/components/SendEvent/SendEventModal', () => ({
  SendEventModal: ({ config }: { config: SendEventConfig }) => {
    mocks.config = config;
    return null;
  },
}));

const renderModal = () =>
  renderToString(<SendEventModal isOpen onClose={() => {}} />);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.config = undefined;
});

describe('event replay key selection', () => {
  it('requests only ordinary event keys from the API', () => {
    mocks.useQuery.mockReturnValue([{ data: undefined }]);
    renderModal();
    const { query, variables } = mocks.useQuery.mock.calls[0][0] as {
      query: DocumentNode;
      variables: { environmentID: string };
    };
    expect(print(query)).toContain('ingestKeys(filter: {source: "key"})');
    expect(variables.environmentID).toBe('environment-id');
  });

  it.each([
    ['no ordinary keys', { data: { environment: { eventKeys: [] } } }],
    ['keys still loading', { data: undefined, fetching: true }],
    ['key query failed', { data: undefined, error: new Error('Unavailable') }],
  ])('does not send when %s', async (_, result) => {
    mocks.useQuery.mockReturnValue([result]);
    renderModal();
    await expect(
      mocks.config!.sendEvent({ name: 'asks/user.signed_ask', data: {} }),
    ).rejects.toThrow('No event key available. Create an event key');
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
