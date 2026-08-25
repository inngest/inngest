import { useEffect, useState } from 'react';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';

import { formatBytes } from '@/components/InfraDashboard/utils';
import { graphql } from '@/gql';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

import { sandboxMetricStats } from './sandboxMetricUtils';
import { formatMemory } from './utils';

const METRIC_LOOKBACK_MS = 10 * 60 * 1_000;
const METRIC_REFRESH_MS = 5_000;

const SandboxMetricsDocument = graphql(`
  query SandboxMetrics(
    $envSlug: String!
    $sandboxID: UUID!
    $from: Time!
    $until: Time!
    $granularitySeconds: Int!
  ) {
    envBySlug(slug: $envSlug) {
      sandboxMetrics(
        id: $sandboxID
        from: $from
        until: $until
        granularitySeconds: $granularitySeconds
      ) {
        metric
        data {
          time
          value
        }
      }
    }
  }
`);

export function SandboxMetrics({
  sandboxID,
  vcpu,
  memoryMB,
}: {
  sandboxID: string;
  vcpu: number;
  memoryMB: number;
}) {
  const environment = useEnvironment();
  const [until, setUntil] = useState(() => new Date());

  useEffect(() => {
    setUntil(new Date());
    const interval = setInterval(() => setUntil(new Date()), METRIC_REFRESH_MS);
    return () => clearInterval(interval);
  }, [sandboxID]);
  const from = new Date(until.getTime() - METRIC_LOOKBACK_MS);

  const { data, error, isLoading, refetch } = useGraphQLQuery({
    query: SandboxMetricsDocument,
    variables: {
      envSlug: environment.slug,
      sandboxID,
      from: from.toISOString(),
      until: until.toISOString(),
      granularitySeconds: 10,
    },
  });
  const stats = sandboxMetricStats(
    data?.envBySlug?.sandboxMetrics ?? [],
    vcpu,
    memoryMB * 1024 * 1024,
  );

  return (
    <section className="mb-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <ResourceStatCard
          label="vCPU"
          current={stats.cpu?.current}
          limit={`${vcpu} vCPU`}
          percent={stats.cpu?.percent}
          formatCurrent={formatVCPU}
          isLoading={isLoading}
        />
        <ResourceStatCard
          label="RAM"
          current={stats.memory?.current}
          limit={formatMemory(memoryMB)}
          percent={stats.memory?.percent}
          peakPercent={stats.memory?.peakPercent}
          peakLabel={
            stats.memory?.peak == null
              ? undefined
              : `${formatBytes(stats.memory.peak)} Peak`
          }
          formatCurrent={formatBytes}
          isLoading={isLoading}
        />
      </div>

      {error && (
        <div className="mt-3">
          <ErrorCard error={error} reset={() => refetch()} />
        </div>
      )}
    </section>
  );
}

function ResourceStatCard({
  label,
  current,
  limit,
  percent,
  peakPercent,
  peakLabel,
  formatCurrent,
  isLoading,
}: {
  label: string;
  current?: number;
  limit: string;
  percent?: number;
  peakPercent?: number | null;
  peakLabel?: string;
  formatCurrent: (value: number) => string;
  isLoading: boolean;
}) {
  const progress = percent ?? 0;

  return (
    <div className="border-muted bg-canvasBase min-h-[98px] rounded-lg border px-5 py-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="text-basis text-sm font-medium">{label}</div>
          <div className="mt-0.5 flex items-end gap-0.5">
            <span className="text-subtle text-[32px] font-medium leading-10 tracking-tight">
              {isLoading || current === undefined
                ? '—'
                : formatCurrent(current)}
            </span>
            <span className="text-light pb-1 text-lg">/{limit}</span>
          </div>
        </div>
        <span className="text-basis mt-9 text-sm">
          {percent === undefined ? '—' : formatPercent(percent)}
        </span>
      </div>
      <div className="relative mt-1">
        <div className="bg-canvasMuted h-1 overflow-hidden rounded-full">
          <div
            className="bg-contrast h-full rounded-full"
            style={{ width: `${progress}%` }}
          />
        </div>
        {peakPercent != null && (
          <div
            className="bg-errorContrast absolute -top-1 h-3 w-px"
            style={{ left: `${peakPercent}%` }}
          />
        )}
        {peakLabel && (
          <span
            className={`text-light absolute top-2 whitespace-nowrap text-xs ${
              peakPercent != null && peakPercent >= 30
                ? '-translate-x-full'
                : ''
            }`}
            style={{ left: `${peakPercent}%` }}
          >
            {peakLabel}
          </span>
        )}
      </div>
    </div>
  );
}

function formatVCPU(value: number): string {
  if (value > 0 && value < 0.01) return '<0.01';
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function formatPercent(value: number): string {
  if (value > 0 && value < 0.1) return '<0.1%';
  return `${value.toFixed(value < 10 ? 1 : 0)}%`;
}
