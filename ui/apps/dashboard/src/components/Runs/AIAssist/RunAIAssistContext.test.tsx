// @vitest-environment jsdom
import { cleanup, render } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { RunAIAssist } from './RunAIAssist';

const mocks = vi.hoisted(() => ({
  state: {
    data: [
      { id: 'other-environment', slug: 'another-production' },
      { id: 'current-environment', slug: 'default-6b963943' },
    ],
    fetching: false,
    error: undefined,
  },
  popover: vi.fn((_props: any) => null),
  refetch: vi.fn(),
}));
vi.mock('@clerk/tanstack-react-start', () => ({
  useOrganization: () => ({ organization: { name: 'Example account' } }),
}));
vi.mock('@tanstack/react-router', () => ({
  useLocation: () => ({ search: {} }),
}));
vi.mock('@/queries/environments', () => ({
  useEnvironments: () => [mocks.state, mocks.refetch],
}));
vi.mock('./tracking', () => ({ useRunAITracking: () => vi.fn() }));
vi.mock('./RunAIAssistPopover', () => ({ RunAIAssistPopover: mocks.popover }));
afterEach(() => {
  cleanup();
  vi.unstubAllEnvs();
});

describe('Runs AI environment context', () => {
  it('resolves the current environment by ID rather than passing its dashboard alias to MCP', () => {
    vi.stubEnv('VITE_API_URL', 'https://api.inngest.com');
    const props = {
      scope: 'env' as const,
      environmentID: 'current-environment',
      environmentSlug: 'production',
      metadataLoading: false,
      metadataError: false,
      onRetry: vi.fn(),
    };
    const { rerender } = render(<RunAIAssist {...props} />);
    expect(mocks.popover.mock.lastCall?.[0]).toMatchObject({
      context: {
        environmentSlug: 'default-6b963943',
        environmentLabel: 'production',
      },
      metadataError: false,
    });
    rerender(<RunAIAssist {...props} environmentID="missing" />);
    expect(mocks.popover.mock.lastCall?.[0]).toMatchObject({
      context: { environmentSlug: '' },
      metadataError: true,
    });
  });
});
