import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import type { FunctionStatusMetricsQuery } from '@/gql/graphql';
import { FailedFunctions } from './FailedFunctions';

const { chart } = vi.hoisted(() => ({ chart: vi.fn(() => null) }));

vi.mock('@inngest/components/Chart/Chart', () => ({ Chart: chart }));
vi.mock('@inngest/components/Button', () => ({ Button: () => null }));
vi.mock('@inngest/components/Info/Info', () => ({ Info: () => null }));
vi.mock('@inngest/components/Link/Link', () => ({ Link: () => null }));
vi.mock('@tanstack/react-router', () => ({ useNavigate: () => vi.fn() }));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => ({ slug: 'production' }),
}));
vi.mock('./FailedRate', () => ({ FailedRate: () => null }));

const metric = (id: string, tagValue: string, value: number) => ({
  id,
  tagName: 'status',
  tagValue,
  data: [{ bucket: '2026-10-28T16:00:00Z', value }],
});

describe('FailedFunctions', () => {
  it('charts individual failed functions even when the overview is grouped by app', () => {
    const workspace = {
      completed: { metrics: [metric('app-a', 'Failed', 10)] },
      completedByFunction: {
        metrics: [
          metric('fn-a', 'Failed', 3),
          metric('fn-b', 'Failed', 7),
          metric('fn-a', 'Completed', 20),
        ],
      },
    } as FunctionStatusMetricsQuery['workspace'];

    renderToStaticMarkup(
      createElement(FailedFunctions, {
        workspace,
        functions: {
          'fn-a': { id: 'fn-a', name: 'Send email' },
          'fn-b': { id: 'fn-b', name: 'Create user' },
        },
      }),
    );

    expect(chart).toHaveBeenCalledWith(
      expect.objectContaining({
        option: expect.objectContaining({
          series: [
            expect.objectContaining({ name: 'Create user', data: [7] }),
            expect.objectContaining({ name: 'Send email', data: [3] }),
          ],
        }),
      }),
      expect.anything(),
    );
  });
});
