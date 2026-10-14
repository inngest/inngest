import { useEffect, useState } from 'react';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';

import { formatBytes } from '@/components/InfraDashboard/utils';
import { graphql } from '@/gql';
import type { SandboxStatus } from '@/gql/graphql';
import { useEnvironment } from '@/components/Environments/environment-context';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

import {
  sandboxLifetimeStats,
  sandboxMetricStats,
  sandboxMetricsWindow,
} from './sandboxMetricUtils';
import { formatDuration, formatMemory } from './utils';

const METRIC_LOOKBACK_MS = 10 * 60 * 1_000;
const METRIC_REFRESH_MS = 5_000;
// metrics.Granularity buckets windows shorter than 3 hours by the minute.
const METRIC_BUCKET_MS = 60 * 1_000;

const SandboxMetricsDocument = graphql(`
  query SandboxMetrics(
    $envSlug: String!
    $sandboxID: UUID!
    $from: Time!
    $until: Time!
  ) {
    envBySlug(slug: $envSlug) {
      cpuUserTime: sandboxMetrics(
        filter: {
          id: $sandboxID
          name: "sandbox_cpu_user_time"
          from: $from
          until: $until
        }
      ) {
        data {
          bucket
          value
        }
      }
      cpuSystemTime: sandboxMetrics(
        filter: {
          id: $sandboxID
          name: "sandbox_cpu_system_time"
          from: $from
          until: $until
        }
      ) {
        data {
          bucket
          value
        }
      }
      memoryCurrent: sandboxMetrics(
        filter: {
          id: $sandboxID
          name: "sandbox_memory_current"
          from: $from
          until: $until
        }
      ) {
        data {
          bucket
          value
        }
      }
      memoryPeak: sandboxMetrics(
        filter: {
          id: $sandboxID
          name: "sandbox_memory_peak"
          from: $from
          until: $until
        }
      ) {
        data {
          bucket
          value
        }
      }
    }
  }
`);

type SandboxMetricsSandbox = {
  id: string;
  status: SandboxStatus;
  vcpu: number;
  memoryMB: number;
  startedAt?: string | null;
  endedAt?: string | null;
  failedAt?: string | null;
  launchUnknownAt?: string | null;
};

export function SandboxMetrics({
  sandbox,
}: {
  sandbox: SandboxMetricsSandbox;
}) {
  const [now] = useState(() => new Date());
  const metricsWindow = sandboxMetricsWindow(sandbox, now);

  switch (metricsWindow.kind) {
    case 'live':
      return <LiveSandboxMetrics sandbox={sandbox} />;
    case 'lifetime':
      return (
        <LifetimeSandboxMetrics
          sandbox={sandbox}
          from={metricsWindow.from}
          until={metricsWindow.until}
          runtimeMS={
            metricsWindow.endedAt.getTime() - metricsWindow.startedAt.getTime()
          }
        />
      );
    case 'pending':
      return (
        <UnavailableSandboxMetrics
          sandbox={sandbox}
          caption="Usage appears once the sandbox is running."
        />
      );
    case 'neverStarted':
      return (
        <UnavailableSandboxMetrics
          sandbox={sandbox}
          caption="This sandbox never started, so no usage was recorded."
        />
      );
    case 'expired':
      return (
        <UnavailableSandboxMetrics
          sandbox={sandbox}
          caption="Usage data is kept for 30 days after a sandbox stops."
        />
      );
  }
}

function LiveSandboxMetrics({ sandbox }: { sandbox: SandboxMetricsSandbox }) {
  const environment = useEnvironment();
  const [until, setUntil] = useState(() => new Date());

  useEffect(() => {
    setUntil(new Date());
    const interval = setInterval(() => setUntil(new Date()), METRIC_REFRESH_MS);
    return () => clearInterval(interval);
  }, [sandbox.id]);
  const from = new Date(until.getTime() - METRIC_LOOKBACK_MS);

  const { data, error, isLoading, refetch } = useGraphQLQuery({
    query: SandboxMetricsDocument,
    variables: {
      envSlug: environment.slug,
      sandboxID: sandbox.id,
      from: from.toISOString(),
      until: until.toISOString(),
    },
  });
  // Each refresh moves the window, so isLoading is true while the previous
  // readings are still on screen; only show the loading state before them.
  const isInitialLoading = isLoading && !data;
  const stats = sandboxMetricStats(
    data?.envBySlug ?? {},
    sandbox.vcpu,
    sandbox.memoryMB * 1024 * 1024,
    { until, bucketMS: METRIC_BUCKET_MS },
  );

  return (
    <MetricsSection
      caption={
        <span className="flex items-center gap-1.5">
          <span className="bg-status-running h-1.5 w-1.5 rounded-full" />
          Live · refreshes every {METRIC_REFRESH_MS / 1_000}s
        </span>
      }
      error={error}
      onRetry={() => refetch()}
    >
      <ResourceStatCard
        label="vCPU"
        description="avg over last minute"
        current={stats.cpu?.current}
        limit={`${sandbox.vcpu} vCPU`}
        percent={stats.cpu?.percent}
        formatCurrent={formatVCPU}
        isLoading={isInitialLoading}
      />
      <ResourceStatCard
        label="RAM"
        description="latest sample"
        current={stats.memory?.current}
        limit={formatMemory(sandbox.memoryMB)}
        percent={stats.memory?.percent}
        peakPercent={stats.memory?.peakPercent}
        peakLabel={
          stats.memory?.peak == null
            ? undefined
            : `${formatBytes(stats.memory.peak)} peak since start`
        }
        formatCurrent={formatBytes}
        isLoading={isInitialLoading}
      />
    </MetricsSection>
  );
}

function LifetimeSandboxMetrics({
  sandbox,
  from,
  until,
  runtimeMS,
}: {
  sandbox: SandboxMetricsSandbox;
  from: Date;
  until: Date;
  runtimeMS: number;
}) {
  const environment = useEnvironment();
  const { data, error, isLoading, refetch } = useGraphQLQuery({
    query: SandboxMetricsDocument,
    variables: {
      envSlug: environment.slug,
      sandboxID: sandbox.id,
      from: from.toISOString(),
      until: until.toISOString(),
    },
  });
  const stats = sandboxLifetimeStats(
    data?.envBySlug ?? {},
    sandbox.vcpu,
    sandbox.memoryMB * 1024 * 1024,
    runtimeMS / 1_000,
  );
  const hasData = stats.cpu !== null || stats.memory !== null;

  return (
    <MetricsSection
      caption={
        !isLoading && !error && !hasData
          ? 'No usage was recorded for this sandbox.'
          : `Lifetime usage · ran for ${formatDuration(runtimeMS)}`
      }
      error={error}
      onRetry={() => refetch()}
    >
      <ResourceStatCard
        label="Avg vCPU"
        current={stats.cpu?.average}
        limit={`${sandbox.vcpu} vCPU`}
        percent={stats.cpu?.percent}
        formatCurrent={formatVCPU}
        isLoading={isLoading}
      />
      <ResourceStatCard
        label="Peak RAM"
        current={stats.memory?.peak}
        limit={formatMemory(sandbox.memoryMB)}
        percent={stats.memory?.percent}
        formatCurrent={formatBytes}
        isLoading={isLoading}
      />
    </MetricsSection>
  );
}

function UnavailableSandboxMetrics({
  sandbox,
  caption,
}: {
  sandbox: SandboxMetricsSandbox;
  caption: string;
}) {
  return (
    <MetricsSection caption={caption}>
      <AllocatedStatCard label="vCPU" allocated={`${sandbox.vcpu} vCPU`} />
      <AllocatedStatCard
        label="RAM"
        allocated={formatMemory(sandbox.memoryMB)}
      />
    </MetricsSection>
  );
}

function MetricsSection({
  caption,
  error,
  onRetry,
  children,
}: {
  caption: React.ReactNode;
  error?: Error;
  onRetry?: () => void;
  children: React.ReactNode;
}) {
  return (
    <section className="mb-6">
      <div className="text-muted mb-2 text-xs">{caption}</div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">{children}</div>

      {error && (
        <div className="mt-3">
          <ErrorCard error={error} reset={() => onRetry?.()} />
        </div>
      )}
    </section>
  );
}

function AllocatedStatCard({
  label,
  allocated,
}: {
  label: string;
  allocated: string;
}) {
  return (
    <div className="border-muted bg-canvasBase min-h-[98px] rounded-lg border px-5 py-4">
      <div className="text-basis text-sm font-medium">{label}</div>
      <div className="mt-0.5 flex items-end gap-1.5">
        <span className="text-light text-[32px] font-medium leading-10 tracking-tight">
          {allocated}
        </span>
        <span className="text-light pb-1 text-lg">allocated</span>
      </div>
    </div>
  );
}

function ResourceStatCard({
  label,
  description,
  current,
  limit,
  percent,
  peakPercent,
  peakLabel,
  formatCurrent,
  isLoading,
}: {
  label: string;
  description?: string;
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
          <div className="text-basis text-sm font-medium">
            {label}
            {description && (
              <span className="text-light font-normal"> · {description}</span>
            )}
          </div>
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
      {/* The peak label is absolutely positioned, so reserve room for it below the bar. */}
      <div className={`relative mt-1 ${peakLabel ? 'pb-5' : ''}`}>
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
