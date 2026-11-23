// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  vi,
} from 'vitest';
import {
  ComputeVerificationStatus as Status,
  ComputeVerificationCheckKind as Kind,
  type ComputeVerificationQuery,
} from '@/gql/graphql';
import { SandboxesAccess } from './SandboxesAccess';
const mocks = vi.hoisted(() => ({
  result: {} as {
    data?: ComputeVerificationQuery;
    fetching?: boolean;
    error?: Error;
  },
  refresh: vi.fn(),
}));
vi.mock('urql', () => ({ useQuery: () => [mocks.result, mocks.refresh] }));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ id: 'env', slug: 'production' }),
}));
vi.mock('./SandboxesList', () => ({
  SandboxesList: () => <div>Sandbox list</div>,
}));
vi.mock('./SandboxesLayout', () => ({
  SandboxesLayout: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock('@/utils/analyticsEvents', () => ({
  trackEmptyStateViewed: vi.fn(),
  trackEmptyStateUpgradeClicked: vi.fn(),
}));
class ResizeObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
}
beforeAll(() => vi.stubGlobal('ResizeObserver', ResizeObserverMock));
afterAll(() => vi.unstubAllGlobals());
afterEach(() => {
  cleanup();
  mocks.result = {};
  vi.clearAllMocks();
});
function setup(status: Status, enabled = false, canVerify = true) {
  mocks.result = {
    fetching: false,
    data: {
      account: {
        id: 'target',
        name: 'Team',
        ents: { sandboxes: { enabled } },
        computeVerification: {
          status,
          canVerify,
          checks: [Kind.Email, Kind.Phone, Kind.Card].map((kind) => ({
            kind,
            passed: false,
            display: null,
          })),
        },
      },
    },
  };
  render(<SandboxesAccess />);
}
describe('sandbox backend eligibility', () => {
  it('lets eligible free accounts start verification instead of upgrading', () => {
    setup(Status.Required);
    expect(
      screen.getByRole('button', { name: 'Start verification' }),
    ).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Upgrade plan' })).toBeNull();
    expect(screen.queryByText('Sandbox list')).toBeNull();
  });
  it('gates a paid entitled account until verification is complete', () => {
    setup(Status.Required, true);
    expect(screen.queryByText('Sandbox list')).toBeNull();
    expect(
      screen.getByRole('button', { name: 'Start verification' }),
    ).toBeTruthy();
  });
  it('requires an admin without offering an upgrade', () => {
    setup(Status.Required, false, false);
    expect(screen.getByText('Ask an admin to verify')).toBeTruthy();
    expect(
      screen.queryByRole('button', { name: 'Start verification' }),
    ).toBeNull();
    expect(screen.queryByRole('button', { name: 'Upgrade plan' })).toBeNull();
  });
  it.each([Status.Verified, Status.Exempt])(
    'shows the sandbox list after %s',
    (status) => {
      setup(status);
      expect(screen.getByText('Sandbox list')).toBeTruthy();
      expect(screen.queryByRole('button')).toBeNull();
    },
  );
  it('preserves an existing entitled account outside verification', () => {
    setup(Status.NotEligible, true);
    expect(screen.getByText('Sandbox list')).toBeTruthy();
  });
  it('preserves upgrades when the backend does not offer verification or access', () => {
    setup(Status.NotEligible);
    expect(
      screen.getAllByRole('button', { name: 'Upgrade plan' }),
    ).toHaveLength(2);
    expect(
      screen.queryByRole('button', { name: 'Start verification' }),
    ).toBeNull();
  });
  it('blocks suspended access even when the sandbox entitlement is enabled', () => {
    setup(Status.Suspended, true);
    expect(
      screen.getByText('Compute and Sandboxes are suspended'),
    ).toBeTruthy();
    expect(screen.queryByText('Sandbox list')).toBeNull();
  });
  it('shows retry after a failed status read instead of inferring an upgrade', () => {
    mocks.result = { fetching: false, error: new Error('offline') };
    render(<SandboxesAccess />);
    expect(screen.getByRole('button', { name: 'Try again' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Upgrade plan' })).toBeNull();
  });
});
