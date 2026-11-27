import { createElement, type PropsWithChildren } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import { concurrencyLinkProps } from '@/components/ConcurrencyAlert/concurrencyAlert';
import { AccountConcurrency } from './AccountConcurrency';

const { useQuery } = vi.hoisted(() => ({ useQuery: vi.fn() }));

vi.mock('urql', () => ({ useQuery }));
vi.mock('@/gql', () => ({ graphql: (query: string) => query }));
vi.mock('@/routes/_authed/billing/route', () => ({
  Route: { useLoaderData: () => ({ profile: { isMarketplace: false } }) },
}));
vi.mock('@inngest/components/Button', () => ({
  Button: ({ href, label }: { href: string; label: string }) => createElement('a', { href }, label),
}));
vi.mock('@inngest/components/Chart/Chart', () => ({ Chart: () => null }));
vi.mock('@inngest/components/Select/Select', () => ({
  Select: Object.assign(() => null, {
    Button: () => null,
    Options: () => null,
    Option: () => null,
  }),
}));
vi.mock('@inngest/components/Tabs/Tabs', () => ({
  default: Object.assign(({ children }: PropsWithChildren) => children, {
    List: () => null,
    Tab: () => null,
  }),
}));

function renderAccountConcurrency(burstLimit: number | null) {
  useQuery.mockImplementation(({ query, variables }) =>
    query.includes('GetAccountConcurrencyTimeSeries')
      ? [
          {
            data: {
              account: {
                concurrencyTimeSeries: {
                  baseLimit: 10,
                  burstLimit,
                  buckets: [],
                  limitHits: [],
                },
              },
            },
            operation: { variables },
          },
        ]
      : [{}],
  );
  return renderToStaticMarkup(createElement(AccountConcurrency));
}

describe('AccountConcurrency', () => {
  it('renders the limited-alert destination without a burst limit', () => {
    const { to } = concurrencyLinkProps({ state: 'limited', surface: 'pill' });
    const markup = renderAccountConcurrency(null);
    const hash = new URL(to, 'https://example.test').hash.slice(1);

    expect(hash).toBe('account-concurrency');
    expect(markup).toContain(`id="${hash}"`);
    expect(markup).toContain('10 steps');
    expect(markup).toContain('Increase concurrency');
    expect(markup).not.toContain('Concurrency during burst');
  });

  it('still shows the burst limit on plans that have one', () => {
    const markup = renderAccountConcurrency(30);

    expect(markup).toContain('Concurrency during burst');
    expect(markup).toContain('30 steps');
  });
});
