import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { defaultLinkStyles } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill';
import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { Table, TextCell } from '@inngest/components/Table';
import { cn } from '@inngest/components/utils/classNames';
import { formatInTimeZone } from '@inngest/components/utils/date';
import { createColumnHelper } from '@tanstack/react-table';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { type GetBurstConcurrencyQuery } from '@/gql/graphql';

const GetBurstConcurrencyDocument = graphql(`
  query GetBurstConcurrency {
    account {
      burstConcurrency {
        limit
        budgetMinutes
        periodEnd
        usage {
          minutesUsed
          minutesRemaining
          bursts {
            startedAt
            endedAt
            peak
            limitHits
          }
        }
      }
    }
  }
`);

type Burst = NonNullable<
  GetBurstConcurrencyQuery['account']['burstConcurrency']
>['usage']['bursts'][number];

const columnHelper = createColumnHelper<Burst>();

function formatMinutes(minutes: number) {
  return `${minutes} ${minutes === 1 ? 'min' : 'mins'}`;
}

function formatBurstWindow({ startedAt, endedAt }: Burst) {
  const start = new Date(startedAt);
  const end = new Date(endedAt);
  const endFormat =
    start.toISOString().slice(0, 10) === end.toISOString().slice(0, 10)
      ? 'HH:mm'
      : 'MMM d HH:mm';
  return `${formatInTimeZone(start, 'UTC', 'MMM d HH:mm')} – ${formatInTimeZone(
    end,
    'UTC',
    endFormat,
  )} UTC`;
}

function durationMinutes({ startedAt, endedAt }: Burst) {
  return Math.round(
    (new Date(endedAt).getTime() - new Date(startedAt).getTime()) / 60_000,
  );
}

export function BurstConcurrency() {
  const [showBreakdown, setShowBreakdown] = useState(true);
  const [{ data, error }] = useQuery({ query: GetBurstConcurrencyDocument });

  if (error) {
    return (
      <div className="mt-10">
        <h3 className="text-basis mb-3 text-xl">Concurrency burst</h3>
        <Alert severity="warning">
          Failed to load burst data. Please{' '}
          <Alert.Link severity="warning" href="https://support.inngest.com">
            contact support
          </Alert.Link>{' '}
          if this does not resolve.
        </Alert>
      </div>
    );
  }

  const burstConcurrency = data?.account.burstConcurrency;
  if (!burstConcurrency) {
    return null;
  }

  const { usage } = burstConcurrency;
  const periodEnd = new Date(burstConcurrency.periodEnd);
  const budgetExhausted = usage.minutesRemaining <= 0;

  const columns = [
    columnHelper.accessor('startedAt', {
      header: 'Burst',
      cell: ({ row }) => <TextCell>{formatBurstWindow(row.original)}</TextCell>,
    }),
    columnHelper.display({
      id: 'duration',
      header: 'Duration',
      cell: ({ row }) => (
        <TextCell>{formatMinutes(durationMinutes(row.original))}</TextCell>
      ),
    }),
    columnHelper.accessor('peak', {
      header: 'Peak concurrency',
      cell: ({ row, getValue }) => (
        <TextCell className={cn(row.original.limitHits > 0 && 'text-error')}>
          {getValue()} of {burstConcurrency.limit}
        </TextCell>
      ),
    }),
    columnHelper.accessor('limitHits', {
      header: 'Limit hits',
      cell: ({ getValue }) => {
        const hits = getValue();
        return (
          <TextCell>{hits === 0 ? 'None' : hits.toLocaleString()}</TextCell>
        );
      },
    }),
  ];

  return (
    <div className="mt-10">
      <h3 className="text-basis mb-3 text-xl">Concurrency burst</h3>
      <p className="text-muted mb-6 text-sm">
        Each month, for one hour, if you exceed your concurrency limit, we
        provide a burst that allows you to process more concurrent steps. This
        boosts your concurrency temporarily and resets every month.
      </p>
      <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-subtle text-sm">Burst usage this month</p>
            <p className="text-basis text-2xl">
              {formatMinutes(usage.minutesUsed)}
            </p>
          </div>
          <Pill className="h-6 px-3 text-sm">
            renews every month on {formatInTimeZone(periodEnd, 'UTC', 'do')}, at{' '}
            {formatInTimeZone(periodEnd, 'UTC', 'h a')} UTC
          </Pill>
        </div>
        <ProgressBar
          className="bg-canvasMuted mt-4 h-2 rounded-full"
          size="small"
          value={usage.minutesUsed}
          limit={burstConcurrency.budgetMinutes}
          kind={budgetExhausted ? 'error' : 'default'}
        />
        <p className="text-subtle mt-2 text-right text-sm">
          {formatMinutes(usage.minutesRemaining)} left
        </p>
        {usage.bursts.length > 0 && (
          <>
            {showBreakdown && (
              <div className="mt-8">
                <Table
                  data={usage.bursts}
                  columns={columns}
                  headerStyle="subtle"
                />
              </div>
            )}
            <button
              type="button"
              aria-expanded={showBreakdown}
              className={cn(defaultLinkStyles, 'mt-4 text-sm')}
              onClick={() => setShowBreakdown((v) => !v)}
            >
              {showBreakdown ? 'Hide breakdown' : 'Show breakdown'}
            </button>
          </>
        )}
      </div>
    </div>
  );
}
