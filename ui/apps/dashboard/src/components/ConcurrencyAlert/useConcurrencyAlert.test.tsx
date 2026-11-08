// @vitest-environment jsdom
import { StrictMode } from 'react';
import { TooltipProvider } from '@inngest/components/Tooltip/Tooltip';
import {
  cleanup,
  fireEvent,
  render,
  renderHook,
  screen,
} from '@testing-library/react';
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  vi,
} from 'vitest';

import { ConcurrencyAlertCard } from './ConcurrencyAlertCard';
import { ConcurrencyAlertPill } from './ConcurrencyAlertPill';
import type {
  ConcurrencyAlertState,
  ConcurrencyAlertSurface,
} from './concurrencyAlert';
import {
  ConcurrencyBurstUsageDocument,
  useConcurrencyAlert,
  type ConcurrencyAlertCheckQuery,
} from './useConcurrencyAlert';

const mocks = vi.hoisted(() => ({ useQuery: vi.fn(), track: vi.fn() }));

vi.mock('@/utils/segment', () => ({ analytics: { track: mocks.track } }));

vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  useLocation: () => ({ href: '/env/production/runs' }),
  Link: ({
    to,
    children,
    onClick,
    'aria-label': ariaLabel,
  }: React.PropsWithChildren<{
    to: string;
    onClick?: React.MouseEventHandler<HTMLAnchorElement>;
    'aria-label'?: string;
  }>) => (
    <a
      href={to}
      aria-label={ariaLabel}
      onClick={(event) => {
        event.preventDefault();
        onClick?.(event);
      }}
    >
      {children}
    </a>
  ),
}));

vi.mock('../ExecutionLimit/ExecutionLimitCard', () => ({
  ExecutionLimitCard: () => <div>execution limit card</div>,
}));
vi.mock('../ExecutionLimit/ExecutionLimitPill', () => ({
  ExecutionLimitPill: () => <div>execution limit pill</div>,
}));

vi.mock('urql', async (importOriginal) => ({
  ...(await importOriginal<typeof import('urql')>()),
  useQuery: mocks.useQuery,
}));

function account({
  bursting,
  limitReached,
}: {
  bursting: boolean;
  limitReached: boolean;
}) {
  return {
    account: {
      id: 'acct',
      burstConcurrency: {
        limit: 300,
        budgetMinutes: 100,
        bursting,
      },
      concurrencyStatus: {
        limitReached,
        minutesWithHits: limitReached ? 3 : 0,
      },
      entitlements: { concurrency: { limit: 100 } },
    },
  };
}

const usage = {
  account: { burstConcurrency: { usage: { minutesRemaining: 40 } } },
};

function mockQueries(data: ConcurrencyAlertCheckQuery | undefined) {
  mocks.useQuery.mockImplementation(({ query }) =>
    query === ConcurrencyBurstUsageDocument
      ? [{ data: data?.account.burstConcurrency?.bursting ? usage : undefined }]
      : [{ data }],
  );
}

class ResizeObserverMock {
  disconnect() {}
  observe() {}
  unobserve() {}
}

beforeAll(() => vi.stubGlobal('ResizeObserver', ResizeObserverMock));
afterAll(() => vi.unstubAllGlobals());

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('useConcurrencyAlert', () => {
  it('reports boosted with the burst limit while bursting', () => {
    mockQueries(account({ bursting: true, limitReached: true }));

    const { result } = renderHook(() => useConcurrencyAlert('pill'));

    expect(result.current?.state).toBe('boosted');
    expect(result.current?.limit).toBe(300);
    expect(result.current?.usedRatio).toBe(0.6);
  });

  it('reports limited with the base limit when not bursting', () => {
    mockQueries(account({ bursting: false, limitReached: true }));

    const { result } = renderHook(() => useConcurrencyAlert('pill'));

    expect(result.current?.state).toBe('limited');
    expect(result.current?.limit).toBe(100);
    expect(mocks.useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        query: ConcurrencyBurstUsageDocument,
        pause: true,
      }),
    );
  });

  it('reports nothing when neither bursting nor limited', () => {
    mockQueries(account({ bursting: false, limitReached: false }));

    const { result } = renderHook(() => useConcurrencyAlert('pill'));

    expect(result.current).toBeNull();
  });

  it('reports limited when burst is off for the account', () => {
    const data = account({ bursting: false, limitReached: true });
    mockQueries({ account: { ...data.account, burstConcurrency: null } });

    const { result } = renderHook(() => useConcurrencyAlert('pill'));

    expect(result.current?.state).toBe('limited');
    expect(result.current?.minutesRemaining).toBeUndefined();
    expect(result.current?.usedRatio).toBe(0);
  });

  it('reports nothing while the query is loading', () => {
    mockQueries(undefined);

    const { result } = renderHook(() => useConcurrencyAlert('pill'));

    expect(result.current).toBeNull();
  });
});

function alerts(surface: ConcurrencyAlertSurface) {
  return (
    <StrictMode>
      <TooltipProvider>
        {surface === 'pill' ? (
          <ConcurrencyAlertPill />
        ) : (
          <ConcurrencyAlertCard collapsed={surface === 'card-collapsed'} />
        )}
      </TooltipProvider>
    </StrictMode>
  );
}

describe('concurrency alert analytics', () => {
  const cases = (['boosted', 'limited'] as const).flatMap((state) =>
    (['pill', 'card', 'card-collapsed'] as const).map((surface) => ({
      state,
      surface,
    })),
  );

  it.each(cases)(
    'pairs $state $surface views and clicks without counting refreshes',
    ({ state, surface }) => {
      const data = account({
        bursting: state === 'boosted',
        limitReached: true,
      });
      // The alert appears before its asynchronous usage query resolves.
      mocks.useQuery.mockImplementation(({ query }) => [
        {
          data: query === ConcurrencyBurstUsageDocument ? undefined : data,
        },
      ]);
      const { rerender } = render(alerts(surface));
      const properties = {
        feature: 'account-concurrency',
        banner_id: 'concurrency-alert',
        account_id: 'acct',
        impression_id: expect.any(String),
        alert_state: state,
        surface,
        cta: state === 'boosted' ? 'view-usage' : 'add-concurrency',
      };
      expect(mocks.track.mock.calls).toEqual([['Banner Viewed', properties]]);
      const impression = mocks.track.mock.calls[0][1];
      expect(impression.impression_id).toBeTruthy();

      mockQueries({ account: { ...data.account } });
      rerender(alerts(surface));
      rerender(alerts(surface));
      expect(mocks.track).toHaveBeenCalledTimes(1);

      const link = screen.getByRole('link');
      expect(link.getAttribute('href')).toBe(
        state === 'boosted'
          ? `/billing/usage?ref=app-concurrency-boost-${surface}#burst-concurrency`
          : `/billing/usage?ref=app-concurrency-limited-${surface}#account-concurrency`,
      );
      fireEvent.click(link);
      expect(mocks.track.mock.calls).toEqual([
        ['Banner Viewed', impression],
        ['Banner CTA Clicked', impression],
      ]);
    },
  );

  it('starts new impressions on state, surface, and account changes, hiding, and remounting', () => {
    let data = account({ bursting: true, limitReached: true });
    mockQueries(data);
    const { rerender, unmount } = render(alerts('card'));
    let previousId: string | undefined;
    function clickAppearance(
      state: ConcurrencyAlertState,
      surface: ConcurrencyAlertSurface,
      accountId = 'acct',
    ) {
      fireEvent.click(screen.getByRole('link'));
      const events = mocks.track.mock.calls.slice(-2);
      expect(events[0]).toEqual([
        'Banner Viewed',
        expect.objectContaining({
          account_id: accountId,
          alert_state: state,
          surface,
          cta: state === 'boosted' ? 'view-usage' : 'add-concurrency',
        }),
      ]);
      expect(events[1]).toEqual(['Banner CTA Clicked', events[0][1]]);
      expect(events[0][1].impression_id).not.toBe(previousId);
      previousId = events[0][1].impression_id;
    }
    clickAppearance('boosted', 'card');

    data = account({ bursting: false, limitReached: true });
    mockQueries(data);
    rerender(alerts('card'));
    clickAppearance('limited', 'card');
    rerender(alerts('card-collapsed'));
    clickAppearance('limited', 'card-collapsed');
    data.account.id = 'other-account';
    mockQueries(data);
    rerender(alerts('card-collapsed'));
    clickAppearance('limited', 'card-collapsed', 'other-account');

    mockQueries(account({ bursting: false, limitReached: false }));
    rerender(alerts('card-collapsed'));
    expect(screen.queryByRole('link')).toBeNull();
    expect(mocks.track).toHaveBeenCalledTimes(8);
    mockQueries(data);
    rerender(alerts('card-collapsed'));
    clickAppearance('limited', 'card-collapsed', 'other-account');
    unmount();
    render(alerts('card-collapsed'));
    clickAppearance('limited', 'card-collapsed', 'other-account');
    expect(mocks.track).toHaveBeenCalledTimes(12);
  });

  it('emits nothing while loading or without an alert', () => {
    mockQueries(undefined);
    const { rerender } = render(alerts('pill'));
    mockQueries(account({ bursting: false, limitReached: false }));
    rerender(alerts('pill'));
    expect(mocks.track).not.toHaveBeenCalled();
  });
});
