import { renderToString } from 'react-dom/server';
import { print, type DocumentNode } from 'graphql';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SendEventConfig } from '@inngest/components/SendEvent/SendEventModal';

import { EnvironmentType } from '@/gql/graphql';
import { SendEventModal } from './SendEventModal';

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
  post: vi.fn(),
  config: undefined as SendEventConfig | undefined,
  environment: { id: 'environment-id', name: 'Production' } as {
    id: string;
    name: string;
    type?: string;
  },
}));

vi.mock('urql', () => ({ useQuery: mocks.useQuery }));
vi.mock('ky', () => ({ default: { post: mocks.post } }));
vi.mock('@tanstack/react-router', () => ({ useRouter: () => ({}) }));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => mocks.environment,
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
  mocks.environment = { id: 'environment-id', name: 'Production' };
});

describe('cURL snippet', () => {
  const eventKeys = {
    data: { environment: { eventKeys: [{ name: 'Default', value: 'key' }] } },
  };

  it('keeps the existing output for payloads without apostrophes', () => {
    mocks.useQuery.mockReturnValue([eventKeys]);
    renderModal();

    expect(
      mocks.config!.generateCurlCode({ name: 'app/user.created', data: {} }),
    ).toMatch(/ \\\n {2}--data '\{"name":"app\/user\.created","data":\{\}\}'$/);
  });

  it('escapes apostrophes in the payload so it stays one shell argument', () => {
    mocks.useQuery.mockReturnValue([eventKeys]);
    renderModal();

    expect(
      mocks.config!.generateCurlCode({
        name: 'app/user.created',
        data: { message: "it's here" },
      }),
    ).toMatch(
      / \\\n {2}--data '\{"name":"app\/user\.created","data":\{"message":"it'\\''s here"\}\}'$/,
    );
  });

  it('quotes the branch environment header as a literal shell word', () => {
    mocks.useQuery.mockReturnValue([eventKeys]);
    mocks.environment = {
      id: 'environment-id',
      name: 'feat/$HOME-"quoted"',
      type: EnvironmentType.BranchChild,
    };
    renderModal();

    expect(
      mocks.config!.generateCurlCode({ name: 'app/user.created', data: {} }),
    ).toContain(` \\\n  -H 'x-inngest-env: feat/$HOME-"quoted"' \\\n`);
  });
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
