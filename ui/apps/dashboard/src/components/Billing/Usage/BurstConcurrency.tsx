import { useState } from 'react';
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
      entitlements: ents {
        concurrency {
          limit
        }
      }
      burstConcurrency {
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
  return `${formatInTimeZone(
    new Date(startedAt),
    'UTC',
    'MMM d HH:mm',
  )} – ${formatInTimeZone(new Date(endedAt), 'UTC', 'HH:mm')} UTC`;
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
    return null;
  }

  const burstConcurrency = data?.account.burstConcurrency;
  if (!burstConcurrency) {
    return null;
  }

  const { usage } = burstConcurrency;
  const accountConcurrencyLimit = data.account.entitlements.concurrency.limit;
  const periodEnd = new Date(burstConcurrency.periodEnd);
  const budgetExhausted = usage.minutesRemaining <= 0;

  const columns = [
    columnHelper.accessor('startedAt', {
      header: 'Burst',
      cell: ({ row }) => (
        <TextCell className="font-normal">
          {formatBurstWindow(row.original)}
        </TextCell>
      ),
    }),
    columnHelper.display({
      id: 'duration',
      header: 'Duration',
      cell: ({ row }) => (
        <TextCell className="font-normal">
          {formatMinutes(durationMinutes(row.original))}
        </TextCell>
      ),
    }),
    columnHelper.accessor('peak', {
      header: 'Peak concurrency',
      cell: ({ getValue }) => (
        <TextCell className="text-error font-normal">
          {getValue()} of {accountConcurrencyLimit}
        </TextCell>
      ),
    }),
    columnHelper.accessor('limitHits', {
      header: 'Runs throttled',
      cell: ({ getValue }) => {
        const hits = getValue();
        return (
          <TextCell className="font-normal">
            {hits === 0 ? 'None' : hits.toLocaleString()}
          </TextCell>
        );
      },
    }),
  ];

  return (
    <div className="mt-10">
      <h3 className="text-basis mb-3 text-xl">Concurrency burst</h3>
      <p className="text-muted mb-6 text-sm">
        Each month, for {formatMinutes(burstConcurrency.budgetMinutes)}, if you
        exceed your concurrency limit, we provide a burst that allows you to
        process more concurrent steps. This boosts your concurrency temporarily
        and resets every month.
      </p>
      <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-basis text-sm">Burst usage this month</p>
            <p className="text-basis text-3xl">
              {formatMinutes(usage.minutesUsed)}
            </p>
          </div>
          <Pill className="h-8 px-4 text-sm">
            renews every month on {formatInTimeZone(periodEnd, 'UTC', 'do')}, at{' '}
            {formatInTimeZone(periodEnd, 'UTC', 'h a')} UTC
          </Pill>
        </div>
        <ProgressBar
          className="bg-canvasMuted mt-4 h-2.5 rounded-full"
          size="small"
          value={usage.minutesUsed}
          limit={burstConcurrency.budgetMinutes}
          kind={budgetExhausted ? 'error' : 'default'}
        />
        <p className="text-basis mt-2 text-right text-sm">
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
                  cellClassName="py-6"
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
