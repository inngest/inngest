// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest';

import { ExecutionLimitBanner } from './ExecutionLimitBanner';
import { ExecutionLimitCard } from './ExecutionLimitCard';
import { ExecutionLimitPill } from './ExecutionLimitPill';

const mocks = vi.hoisted(() => ({
  useSkippableGraphQLQuery: vi.fn(),
}));

vi.mock('@/components/FeatureFlags/hooks', () => ({
  useBooleanFlag: () => ({ value: true, isReady: true }),
}));

vi.mock('@/gql', () => ({
  graphql: (source: string) => source,
}));

vi.mock('@/utils/useGraphQLQuery', () => ({
  useSkippableGraphQLQuery: mocks.useSkippableGraphQLQuery,
}));

vi.mock('@/utils/urls', () => ({
  pathCreator: {
    billing: ({ ref }: { ref: string }) => `/billing?ref=${ref}`,
  },
}));

vi.mock('@inngest/components/Link', () => ({
  Link: ({ to, children }: React.PropsWithChildren<{ to: string }>) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock('@inngest/components/Menu/MenuItem', () => ({
  MenuItem: ({ to, text }: { to: string; text: string }) => (
    <a href={to}>{text}</a>
  ),
}));

class ResizeObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
}

beforeAll(() => vi.stubGlobal('ResizeObserver', ResizeObserverMock));
afterAll(() => vi.unstubAllGlobals());

function mockCap(cap: {
  usage: number;
  limit: number;
  overageLimit: number;
  enforced: boolean;
  exceeded: boolean;
}) {
  mocks.useSkippableGraphQLQuery.mockImplementation(
    ({ skip }: { skip: boolean }) => ({
      data: skip
        ? undefined
        : {
            account: {
              id: 'account',
              marketplace: null,
              marketplaceBillingURL: null,
              executionCap: cap,
            },
          },
    }),
  );
}

const grace = {
  usage: 52_000,
  limit: 50_000,
  overageLimit: 10_000,
  enforced: true,
  exceeded: false,
};

beforeEach(() => {
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('grace period', () => {
  beforeEach(() => mockCap(grace));

  it('shows the grace period in the pill', () => {
    render(<ExecutionLimitPill />);

    expect(screen.getByText(/grace period/)).toBeTruthy();
    expect(screen.getByText(/8,000 executions left/)).toBeTruthy();
    expect(
      screen.getByRole('link', { name: 'Upgrade' }).getAttribute('href'),
    ).toBe('/billing?ref=app-hobby-execution-grace-pill');
  });

  it('shows the grace period in the card without a dismiss button', () => {
    render(<ExecutionLimitCard collapsed={false} />);

    expect(screen.getByText('Temporary grace period')).toBeTruthy();
    expect(screen.getByText(/52,000 executions this month/)).toBeTruthy();
    expect(screen.getByText(/8,000 executions remaining/)).toBeTruthy();
    expect(screen.queryByText(/paused/)).toBeNull();
    expect(screen.queryByLabelText('Dismiss for 24 hours')).toBeNull();
    expect(
      screen.getByRole('link', { name: 'Upgrade plan' }).getAttribute('href'),
    ).toBe('/billing?ref=app-hobby-execution-grace-card');
  });

  it('shows an upgrade link in the collapsed card', () => {
    render(<ExecutionLimitCard collapsed />);

    expect(
      screen.getByRole('link', { name: 'Upgrade plan' }).getAttribute('href'),
    ).toBe('/billing?ref=app-hobby-execution-grace-card-collapsed');
  });

  it('shows the grace period in the runs banner', () => {
    render(<ExecutionLimitBanner />);

    expect(screen.getByText(/grace period/)).toBeTruthy();
    expect(screen.getByText(/8,000 executions left/)).toBeTruthy();
    expect(
      screen.getByRole('link', { name: 'Upgrade' }).getAttribute('href'),
    ).toBe('/billing?ref=app-hobby-execution-grace-banner');
  });
});

describe('limit exceeded without enforcement', () => {
  beforeEach(() =>
    mockCap({
      usage: 60_000,
      limit: 50_000,
      overageLimit: 0,
      enforced: false,
      exceeded: true,
    }),
  );

  it('never says runs are paused', () => {
    render(
      <>
        <ExecutionLimitPill />
        <ExecutionLimitCard collapsed={false} />
        <ExecutionLimitBanner />
      </>,
    );

    expect(screen.queryByText(/are paused/)).toBeNull();
    expect(screen.queryByText(/limit reached/)).toBeNull();
    expect(screen.getByText(/almost reached/)).toBeTruthy();
  });
});

describe('limit exceeded and enforced', () => {
  beforeEach(() =>
    mockCap({ ...grace, usage: 60_000, enforced: true, exceeded: true }),
  );

  it('says new runs are paused on the card and banner', () => {
    render(
      <>
        <ExecutionLimitPill />
        <ExecutionLimitCard collapsed={false} />
        <ExecutionLimitBanner />
      </>,
    );

    expect(screen.getByText('New runs are paused')).toBeTruthy();
    expect(screen.getByText(/^New runs are paused\. You've used/)).toBeTruthy();
    expect(
      screen.getByText(/Hobby account executions limit reached/),
    ).toBeTruthy();
    expect(screen.queryByText(/grace period/)).toBeNull();
  });
});
