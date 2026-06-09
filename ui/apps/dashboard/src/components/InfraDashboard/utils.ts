import type {
  GetFunctionsQuery,
  GetFunctionsUsageQuery,
  MetricsData,
  TimeSeriesPoint,
} from '@/gql/graphql';
import type { Function as FunctionRow } from '@inngest/components/types/function';
import type { InfraPlan, InfraPlanSku, InfraTierId } from './placeholderData';

export type DashboardKpi = {
  label: string;
  value: string;
  delta?: string;
  progress?: number;
};

export type TopFunctionRow = FunctionRow;

type WorkflowUsage =
  GetFunctionsUsageQuery['workspace']['workflows']['data'][number];

type WorkflowSummary =
  GetFunctionsQuery['workspace']['workflows']['data'][number];

export function formatCompactNumber(value: number): string {
  if (!Number.isFinite(value)) {
    return '0';
  }

  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: value >= 1_000_000 ? 1 : 0,
    notation: value >= 10_000 ? 'compact' : 'standard',
  }).format(value);
}

export function formatPercent(value: number): string {
  return `${Math.round(value)}%`;
}

export function formatDeltaPercent(value: number): string {
  return `${value > 0 ? '+' : ''}${value.toFixed(1)}%`;
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0B';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const formatted =
    value >= 100 || Number.isInteger(value)
      ? value.toFixed(0)
      : value.toFixed(1);
  return `${formatted}${units[unitIndex]}`;
}

export function formatCentsMonthly(
  amountCents: number | null | undefined,
): string {
  if (amountCents === null || amountCents === undefined) {
    return '';
  }

  return new Intl.NumberFormat('en-US', {
    currency: 'USD',
    maximumFractionDigits: 0,
    style: 'currency',
  }).format(amountCents / 100);
}

type BillingPlanSource = {
  amount?: number | null;
  entitlements?: {
    concurrency?: { limit?: number | null } | null;
    events?: { limit?: number | null } | null;
    functionBacklogSize?: { limit?: number | null } | null;
  } | null;
  isFree?: boolean | null;
  name?: string | null;
  slug?: string | null;
};

type AccountEntitlementsSource = {
  concurrency?: { limit?: number | null } | null;
  events?: { limit?: number | null } | null;
  functionBacklogSize?: { limit?: number | null } | null;
};

function preferDefinedLimit<T extends number | null>(
  accountLimit: T | undefined,
  planLimit: T | undefined,
): T | undefined {
  return accountLimit !== undefined ? accountLimit : planLimit;
}

export function inferInfraPlanSku({
  concurrencyLimit,
  defaultSku,
  plan,
  plans,
}: {
  concurrencyLimit?: number | null;
  defaultSku: InfraPlanSku;
  plan?: BillingPlanSource | null;
  plans: InfraPlan[];
}): InfraPlanSku {
  if (typeof concurrencyLimit === 'number') {
    const sortedPlans = [...plans].sort(
      (a, b) => a.execConcurrencyLimit - b.execConcurrencyLimit,
    );
    const flooredPlan = sortedPlans.reduce<InfraPlan | undefined>(
      (best, candidate) =>
        candidate.execConcurrencyLimit <= concurrencyLimit ? candidate : best,
      undefined,
    );

    return flooredPlan?.sku ?? sortedPlans[0]?.sku ?? defaultSku;
  }

  const planKey = `${plan?.slug ?? ''} ${plan?.name ?? ''}`.toLowerCase();

  if (plan?.isFree || planKey.match(/\b(hobby|free|in-xs|xs)\b/)) {
    return 'IN-XS';
  }

  if (planKey.match(/\b(pro|basic|in-s)\b/)) {
    return 'IN-S';
  }

  if (planKey.match(/\b(enterprise|in-xl|xl)\b/)) {
    return 'IN-XL';
  }

  return defaultSku;
}

export function getCurrentInfraTierId(planSku: InfraPlanSku): InfraTierId {
  return planSku === 'IN-XS' ? 'free' : 'shared';
}

export function mergeBillingPlanIntoInfraPlans({
  accountEntitlements,
  defaultSku,
  plan,
  plans,
}: {
  accountEntitlements?: AccountEntitlementsSource | null;
  defaultSku: InfraPlanSku;
  plan?: BillingPlanSource | null;
  plans: InfraPlan[];
}): {
  currentPlan: InfraPlan;
  currentPlanSku: InfraPlanSku;
  plans: InfraPlan[];
} {
  const concurrencyLimit =
    accountEntitlements?.concurrency?.limit ??
    plan?.entitlements?.concurrency?.limit ??
    null;
  const eventLimit = preferDefinedLimit(
    accountEntitlements?.events?.limit,
    plan?.entitlements?.events?.limit,
  );
  const queueDepthLimit = preferDefinedLimit(
    accountEntitlements?.functionBacklogSize?.limit,
    plan?.entitlements?.functionBacklogSize?.limit,
  );
  const currentPlanSku = inferInfraPlanSku({
    concurrencyLimit,
    defaultSku,
    plan,
    plans,
  });
  const fallbackPlan =
    plans.find((candidate) => candidate.sku === currentPlanSku) ?? plans[0];
  const hasLiveEntitlements =
    typeof concurrencyLimit === 'number' ||
    eventLimit !== undefined ||
    queueDepthLimit !== undefined;

  if (!plan && !hasLiveEntitlements) {
    return {
      currentPlan: fallbackPlan,
      currentPlanSku: fallbackPlan.sku,
      plans,
    };
  }

  const currentPlan: InfraPlan = {
    ...fallbackPlan,
    eventStream:
      eventLimit === null
        ? 'Unlimited events/mo'
        : eventLimit === undefined
        ? fallbackPlan.eventStream
        : `${formatCompactNumber(eventLimit)} events/mo`,
    eventStreamLimit:
      eventLimit === undefined ? fallbackPlan.eventStreamLimit : eventLimit,
    eventStreamUnit:
      eventLimit === undefined ? fallbackPlan.eventStreamUnit : 'events',
    execConcurrency:
      typeof concurrencyLimit === 'number'
        ? formatCompactNumber(concurrencyLimit)
        : fallbackPlan.execConcurrency,
    execConcurrencyLimit:
      typeof concurrencyLimit === 'number'
        ? concurrencyLimit
        : fallbackPlan.execConcurrencyLimit,
    isCurrent: true,
    priceMonthly: formatCentsMonthly(plan?.amount) || fallbackPlan.priceMonthly,
    queueDepth:
      queueDepthLimit === null
        ? 'Unlimited'
        : queueDepthLimit === undefined
        ? fallbackPlan.queueDepth
        : formatCompactNumber(queueDepthLimit),
    queueDepthLimit:
      queueDepthLimit === undefined
        ? fallbackPlan.queueDepthLimit
        : queueDepthLimit,
  };

  return {
    currentPlan,
    currentPlanSku,
    plans: plans.map((candidate) =>
      candidate.sku === currentPlanSku ? currentPlan : candidate,
    ),
  };
}

export function daysUntil(
  dateString?: string | null,
  now = new Date(),
): number | null {
  if (!dateString) {
    return null;
  }

  const date = new Date(dateString);
  if (Number.isNaN(date.valueOf())) {
    return null;
  }

  const diffMs = date.valueOf() - now.valueOf();
  return Math.max(0, Math.ceil(diffMs / 86_400_000));
}

export function utcEndOfMonth(now = new Date()): Date {
  return new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0, 23, 59, 59, 999),
  );
}

export function billingCycleDaysRemaining(
  nextInvoiceDate?: string | null,
  now = new Date(),
): number {
  return (
    daysUntil(nextInvoiceDate, now) ??
    daysUntil(utcEndOfMonth(now).toISOString(), now) ??
    0
  );
}

export function getUtcMonthToDateRange(now = new Date()) {
  return {
    from: new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1)),
    month: now.getUTCMonth() + 1,
    until: now,
    year: now.getUTCFullYear(),
  };
}

export function sumMetricValues(
  metrics: Array<{ data: Array<Pick<MetricsData, 'value'>> }> | undefined,
): number {
  return (
    metrics?.reduce((total, metric) => {
      return total + sumDataValues(metric.data);
    }, 0) ?? 0
  );
}

export function latestMetricTotal(
  metrics: Array<{ data: Array<Pick<MetricsData, 'value'>> }> | undefined,
): number {
  return (
    metrics?.reduce((total, metric) => {
      return total + (metric.data.at(-1)?.value ?? 0);
    }, 0) ?? 0
  );
}

export function latestBucketMetricTotal(
  metrics:
    | Array<{ data: Array<Pick<MetricsData, 'bucket' | 'value'>> }>
    | undefined,
): number {
  const latestBucket = metrics?.reduce<string | undefined>((latest, metric) => {
    return metric.data.reduce<string | undefined>((innerLatest, point) => {
      if (!innerLatest || Date.parse(point.bucket) > Date.parse(innerLatest)) {
        return point.bucket;
      }

      return innerLatest;
    }, latest);
  }, undefined);

  if (!metrics || !latestBucket) {
    return 0;
  }

  return metrics.reduce((total, metric) => {
    return (
      total +
      metric.data.reduce((metricTotal, point) => {
        return point.bucket === latestBucket
          ? metricTotal + point.value
          : metricTotal;
      }, 0)
    );
  }, 0);
}

export function sumTimeSeriesValues(
  series: Array<{ data: Array<Pick<TimeSeriesPoint, 'value'>> }> | undefined,
): number {
  return (
    series?.reduce((total, item) => {
      return (
        total +
        item.data.reduce((itemTotal, point) => {
          return itemTotal + (point.value ?? 0);
        }, 0)
      );
    }, 0) ?? 0
  );
}

export function sumDataValues(
  data: Array<Pick<MetricsData, 'value'>> | undefined,
): number {
  return data?.reduce((total, point) => total + point.value, 0) ?? 0;
}

export function calculateUsageShare(value: number, total: number): number {
  if (!total) {
    return 0;
  }

  return Math.round((value / total) * 1000) / 10;
}

export function buildTopFunctionRows({
  functions,
  usage,
}: {
  functions: WorkflowSummary[] | undefined;
  usage: WorkflowUsage[] | undefined;
}): TopFunctionRow[] {
  const summariesBySlug = new Map(
    functions?.map((fn) => [fn.slug, fn] as const) ?? [],
  );

  return (
    usage
      ?.map((fn) => {
        const summary = summariesBySlug.get(fn.slug);
        const dailyFailureCount = fn.dailyFailures.total;
        const dailyFinishedCount =
          fn.dailyCompleted.total + fn.dailyCancelled.total + dailyFailureCount;
        const failureRate = dailyFinishedCount
          ? Math.round((dailyFailureCount / dailyFinishedCount) * 10000) / 100
          : 0;

        return {
          app: summary?.app
            ? {
                externalID: summary.app.externalID,
                name: summary.app.name,
              }
            : undefined,
          failureRate,
          id: summary?.id ?? fn.id,
          isArchived: summary?.isArchived,
          isPaused: summary?.isPaused,
          name: summary?.name ?? fn.slug,
          slug: fn.slug,
          triggers:
            summary?.triggers.map((trigger) => ({
              type: trigger.type as TopFunctionRow['triggers'][number]['type'],
              value: trigger.value,
            })) ?? [],
          usage: {
            dailyVolumeSlots: fn.dailyStarts.data.map((usageSlot, index) => ({
              failureCount: fn.dailyFailures.data[index]?.count ?? 0,
              startCount: usageSlot.count,
            })),
            totalVolume: fn.dailyStarts.total,
          },
        };
      })
      .sort((a, b) => (b.usage?.totalVolume ?? 0) - (a.usage?.totalVolume ?? 0))
      .slice(0, 5) ?? []
  );
}
