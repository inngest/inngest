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

type PlanPriceSource = {
  amount?: number | null;
  isFree?: boolean | null;
  isLegacy?: boolean | null;
  name?: string | null;
} | null;

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

export type BillingPlanSource = {
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

export const FREE_INFRA_PLAN_SLUG = 'hobby-free-2025-08-08';
export const PRO_INFRA_PLAN_SLUG = 'pro-2025-08-08';

const PRO_PLAN_BASE_AMOUNT_CENTS = 7_500;

export type InfraConcurrencyAddonSource = {
  available?: boolean | null;
  baseValue?: number | null;
  maxValue?: number | null;
  name?: string | null;
  price?: number | null;
  purchaseCount?: number | null;
  quantityPer?: number | null;
};

export type InfraPlanCheckoutItem = {
  amount: number;
  name: string;
  planSlug: string;
  quantity: number;
};

export type InfraPlanAddonUpdate = {
  addonName: string;
  addonQuantity: number;
  estimatedMonthlyAddonCost: number;
  isIncrease: boolean;
  targetConcurrency: number;
  targetMonthlyAmountCents: number;
  targetSku: InfraPlanSku;
};

export type InfraPlanBillingAction =
  | { type: 'current' }
  | {
      addonUpdate: InfraPlanAddonUpdate | null;
      item: InfraPlanCheckoutItem;
      type: 'cancel-to-free';
    }
  | {
      addonUpdate: InfraPlanAddonUpdate | null;
      item: InfraPlanCheckoutItem;
      type: 'upgrade-base-plan';
    }
  | (InfraPlanAddonUpdate & { type: 'update-concurrency-addon' })
  | { reason: string; type: 'unavailable' };

const INFRA_PLAN_BILLING_TARGETS: Record<
  InfraPlanSku,
  {
    basePlanSlug: string;
    monthlyAmountCents: number;
    targetConcurrency: number;
  }
> = {
  'IN-XS': {
    basePlanSlug: FREE_INFRA_PLAN_SLUG,
    monthlyAmountCents: 0,
    targetConcurrency: 5,
  },
  'IN-S': {
    basePlanSlug: PRO_INFRA_PLAN_SLUG,
    monthlyAmountCents: 9_900,
    targetConcurrency: 100,
  },
  'IN-M': {
    basePlanSlug: PRO_INFRA_PLAN_SLUG,
    monthlyAmountCents: 24_900,
    targetConcurrency: 250,
  },
  'IN-L': {
    basePlanSlug: PRO_INFRA_PLAN_SLUG,
    monthlyAmountCents: 59_900,
    targetConcurrency: 500,
  },
  'IN-XL': {
    basePlanSlug: PRO_INFRA_PLAN_SLUG,
    monthlyAmountCents: 119_900,
    targetConcurrency: 1_000,
  },
};

const CONCURRENCY_ADDON_NAME = 'concurrency';
const PRO_BASE_CONCURRENCY =
  INFRA_PLAN_BILLING_TARGETS['IN-S'].targetConcurrency;
const DEFAULT_CONCURRENCY_ADDON_QUANTITY_PER = 100;

function isUsableConcurrencyAddon(
  addon?: InfraConcurrencyAddonSource | null,
): boolean {
  return Boolean(
    addon?.name &&
      typeof addon.maxValue === 'number' &&
      typeof addon.price === 'number' &&
      typeof addon.quantityPer === 'number' &&
      addon.quantityPer > 0,
  );
}

export function pickInfraConcurrencyAddon({
  accountAddon,
  planAddon,
}: {
  accountAddon?: InfraConcurrencyAddonSource | null;
  planAddon?: InfraConcurrencyAddonSource | null;
}): InfraConcurrencyAddonSource | null | undefined {
  if (isUsableConcurrencyAddon(accountAddon)) {
    return accountAddon;
  }

  if (isUsableConcurrencyAddon(planAddon)) {
    return planAddon;
  }

  return accountAddon ?? planAddon;
}

function isFreeBillingPlan(plan?: BillingPlanSource | null): boolean {
  return Boolean(plan?.isFree || plan?.slug === FREE_INFRA_PLAN_SLUG);
}

function isMappedInfraBillingPlan(plan?: BillingPlanSource | null): boolean {
  return (
    plan?.slug === FREE_INFRA_PLAN_SLUG || plan?.slug === PRO_INFRA_PLAN_SLUG
  );
}

function buildCheckoutItem(planSlug: string): InfraPlanCheckoutItem {
  if (planSlug === FREE_INFRA_PLAN_SLUG) {
    return {
      amount: 0,
      name: 'Hobby',
      planSlug,
      quantity: 1,
    };
  }

  return {
    amount: PRO_PLAN_BASE_AMOUNT_CENTS,
    name: 'Pro',
    planSlug,
    quantity: 1,
  };
}

function getTargetMonthlyAmountCents({
  proPlanAmountCents,
  targetSku,
}: {
  proPlanAmountCents?: number | null;
  targetSku: InfraPlanSku;
}): number {
  if (targetSku === 'IN-S' && typeof proPlanAmountCents === 'number') {
    return proPlanAmountCents;
  }

  return INFRA_PLAN_BILLING_TARGETS[targetSku].monthlyAmountCents;
}

function buildProCheckoutItem(
  proPlanAmountCents?: number | null,
): InfraPlanCheckoutItem {
  return {
    ...buildCheckoutItem(PRO_INFRA_PLAN_SLUG),
    amount:
      typeof proPlanAmountCents === 'number'
        ? proPlanAmountCents
        : PRO_PLAN_BASE_AMOUNT_CENTS,
  };
}

function buildAddonUpdate({
  addon,
  currentConcurrencyLimit,
  targetConcurrency,
  targetMonthlyAmountCents,
  targetSku,
}: {
  addon?: InfraConcurrencyAddonSource | null;
  currentConcurrencyLimit?: number | null;
  targetConcurrency: number;
  targetMonthlyAmountCents: number;
  targetSku: InfraPlanSku;
}): InfraPlanAddonUpdate | { reason: string; type: 'unavailable' } {
  if (!addon) {
    return {
      reason: 'Concurrency add-on metadata is unavailable.',
      type: 'unavailable',
    };
  }

  if (!addon.name) {
    return {
      reason: 'Concurrency add-on name is missing.',
      type: 'unavailable',
    };
  }

  if (typeof addon.price !== 'number') {
    return {
      reason: 'Concurrency add-on price is missing.',
      type: 'unavailable',
    };
  }

  if (typeof addon.quantityPer !== 'number' || addon.quantityPer <= 0) {
    return {
      reason: 'Concurrency add-on sizing is unavailable.',
      type: 'unavailable',
    };
  }

  if (
    typeof addon.maxValue === 'number' &&
    targetConcurrency > addon.maxValue
  ) {
    return {
      reason: 'Selected concurrency is above the add-on maximum.',
      type: 'unavailable',
    };
  }

  const addonQuantity = Math.max(
    0,
    Math.ceil((targetConcurrency - PRO_BASE_CONCURRENCY) / addon.quantityPer),
  );

  return {
    addonName: addon.name,
    addonQuantity,
    estimatedMonthlyAddonCost: addonQuantity * addon.price,
    isIncrease:
      typeof currentConcurrencyLimit !== 'number' ||
      targetConcurrency > currentConcurrencyLimit,
    targetConcurrency,
    targetMonthlyAmountCents,
    targetSku,
  };
}

function buildAddonRemoval({
  addon,
  currentConcurrencyLimit,
  targetConcurrency,
  targetMonthlyAmountCents,
  targetSku,
}: {
  addon?: InfraConcurrencyAddonSource | null;
  currentConcurrencyLimit?: number | null;
  targetConcurrency: number;
  targetMonthlyAmountCents: number;
  targetSku: InfraPlanSku;
}): InfraPlanAddonUpdate | null {
  if (
    typeof currentConcurrencyLimit !== 'number' ||
    (currentConcurrencyLimit <= PRO_BASE_CONCURRENCY &&
      (addon?.purchaseCount ?? 0) <= 0)
  ) {
    return null;
  }

  return {
    addonName: addon?.name ?? CONCURRENCY_ADDON_NAME,
    addonQuantity: 0,
    estimatedMonthlyAddonCost: 0,
    isIncrease: false,
    targetConcurrency,
    targetMonthlyAmountCents,
    targetSku,
  };
}

function buildStaticAddonUpdate({
  currentConcurrencyLimit,
  proPlanAmountCents,
  targetConcurrency,
  targetMonthlyAmountCents,
  targetSku,
}: {
  currentConcurrencyLimit?: number | null;
  proPlanAmountCents?: number | null;
  targetConcurrency: number;
  targetMonthlyAmountCents: number;
  targetSku: InfraPlanSku;
}): InfraPlanAddonUpdate {
  const addonQuantity = Math.max(
    0,
    Math.ceil(
      (targetConcurrency - PRO_BASE_CONCURRENCY) /
        DEFAULT_CONCURRENCY_ADDON_QUANTITY_PER,
    ),
  );

  return {
    addonName: CONCURRENCY_ADDON_NAME,
    addonQuantity,
    estimatedMonthlyAddonCost: Math.max(
      0,
      targetMonthlyAmountCents -
        (proPlanAmountCents ?? PRO_PLAN_BASE_AMOUNT_CENTS),
    ),
    isIncrease:
      typeof currentConcurrencyLimit !== 'number' ||
      targetConcurrency > currentConcurrencyLimit,
    targetConcurrency,
    targetMonthlyAmountCents,
    targetSku,
  };
}

export function getInfraPlanBillingAction({
  concurrencyAddon,
  currentConcurrencyLimit,
  currentPlan,
  currentPlanSku,
  proPlanAmountCents,
  targetSku,
}: {
  concurrencyAddon?: InfraConcurrencyAddonSource | null;
  currentConcurrencyLimit?: number | null;
  currentPlan?: BillingPlanSource | null;
  currentPlanSku: InfraPlanSku;
  proPlanAmountCents?: number | null;
  targetSku: InfraPlanSku;
}): InfraPlanBillingAction {
  if (!currentPlan) {
    return { reason: 'Billing plan is still loading.', type: 'unavailable' };
  }

  if (isMappedInfraBillingPlan(currentPlan) && targetSku === currentPlanSku) {
    return { type: 'current' };
  }

  const target = INFRA_PLAN_BILLING_TARGETS[targetSku];
  const targetMonthlyAmountCents = getTargetMonthlyAmountCents({
    proPlanAmountCents,
    targetSku,
  });
  const currentIsFree = isFreeBillingPlan(currentPlan);
  const currentIsMappedInfraPlan = isMappedInfraBillingPlan(currentPlan);

  if (target.basePlanSlug === FREE_INFRA_PLAN_SLUG) {
    return currentPlan.slug === FREE_INFRA_PLAN_SLUG
      ? { type: 'current' }
      : {
          addonUpdate: buildAddonRemoval({
            addon: concurrencyAddon,
            currentConcurrencyLimit,
            targetConcurrency: target.targetConcurrency,
            targetMonthlyAmountCents,
            targetSku,
          }),
          item: buildCheckoutItem(FREE_INFRA_PLAN_SLUG),
          type: 'cancel-to-free',
        };
  }

  if (target.targetConcurrency <= PRO_BASE_CONCURRENCY) {
    if (currentIsFree || currentPlan?.slug !== PRO_INFRA_PLAN_SLUG) {
      return {
        addonUpdate: null,
        item: buildProCheckoutItem(proPlanAmountCents),
        type: 'upgrade-base-plan',
      };
    }

    return {
      addonName: concurrencyAddon?.name ?? CONCURRENCY_ADDON_NAME,
      addonQuantity: 0,
      estimatedMonthlyAddonCost: 0,
      isIncrease:
        typeof currentConcurrencyLimit !== 'number' ||
        target.targetConcurrency > currentConcurrencyLimit,
      targetConcurrency: target.targetConcurrency,
      targetMonthlyAmountCents,
      targetSku,
      type: 'update-concurrency-addon',
    };
  }

  const addonUpdate = buildAddonUpdate({
    addon: concurrencyAddon,
    currentConcurrencyLimit,
    targetConcurrency: target.targetConcurrency,
    targetMonthlyAmountCents,
    targetSku,
  });
  const resolvedAddonUpdate =
    'type' in addonUpdate && !currentIsMappedInfraPlan
      ? buildStaticAddonUpdate({
          currentConcurrencyLimit,
          proPlanAmountCents,
          targetConcurrency: target.targetConcurrency,
          targetMonthlyAmountCents,
          targetSku,
        })
      : addonUpdate;

  if ('type' in resolvedAddonUpdate) {
    return resolvedAddonUpdate;
  }

  if (currentIsFree || currentPlan?.slug !== PRO_INFRA_PLAN_SLUG) {
    return {
      addonUpdate:
        resolvedAddonUpdate.addonQuantity > 0 ? resolvedAddonUpdate : null,
      item: buildProCheckoutItem(proPlanAmountCents),
      type: 'upgrade-base-plan',
    };
  }

  return {
    ...resolvedAddonUpdate,
    type: 'update-concurrency-addon',
  };
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

export function isEnterprisePlanName(planName?: string | null): boolean {
  return planName?.toLowerCase().includes('enterprise') ?? false;
}

export function pickCheapestEnabledProPlanAmount(
  plans?: PlanPriceSource[],
): number | null {
  const proPlans =
    plans?.filter(
      (plan) =>
        plan &&
        !plan.isFree &&
        !plan.isLegacy &&
        plan.name?.toLowerCase().includes('pro') &&
        typeof plan.amount === 'number',
    ) ?? [];

  return (
    proPlans.sort((a, b) => {
      return (a?.amount ?? Infinity) - (b?.amount ?? Infinity);
    })[0]?.amount ?? null
  );
}

function applyProPlanAmountToInfraPlans({
  plans,
  proPlanAmountCents,
}: {
  plans: InfraPlan[];
  proPlanAmountCents?: number | null;
}): InfraPlan[] {
  if (typeof proPlanAmountCents !== 'number') {
    return plans;
  }

  return plans.map((plan) =>
    plan.sku === 'IN-S'
      ? {
          ...plan,
          priceMonthly: formatCentsMonthly(proPlanAmountCents),
        }
      : plan,
  );
}

export function mergeBillingPlanIntoInfraPlans({
  accountEntitlements,
  defaultSku,
  plan,
  plans,
  proPlanAmountCents,
}: {
  accountEntitlements?: AccountEntitlementsSource | null;
  defaultSku: InfraPlanSku;
  plan?: BillingPlanSource | null;
  plans: InfraPlan[];
  proPlanAmountCents?: number | null;
}): {
  currentPlan: InfraPlan;
  currentPlanSku: InfraPlanSku;
  plans: InfraPlan[];
} {
  const pricedPlans = applyProPlanAmountToInfraPlans({
    plans,
    proPlanAmountCents,
  });
  const concurrencyLimit =
    accountEntitlements?.concurrency?.limit ??
    plan?.entitlements?.concurrency?.limit ??
    null;
  const currentPlanSku = inferInfraPlanSku({
    concurrencyLimit,
    defaultSku,
    plan,
    plans: pricedPlans,
  });
  const fallbackPlan =
    pricedPlans.find((candidate) => candidate.sku === currentPlanSku) ??
    pricedPlans[0];
  const hasLiveEntitlements = typeof concurrencyLimit === 'number';
  const hasMappedBillingPlan = isMappedInfraBillingPlan(plan);

  if (!plan && !hasLiveEntitlements) {
    const currentFallbackPlan = { ...fallbackPlan, isCurrent: true };

    return {
      currentPlan: currentFallbackPlan,
      currentPlanSku: fallbackPlan.sku,
      plans: pricedPlans.map((candidate) =>
        candidate.sku === currentFallbackPlan.sku
          ? currentFallbackPlan
          : { ...candidate, isCurrent: false },
      ),
    };
  }

  const currentPlan: InfraPlan = {
    ...fallbackPlan,
    execConcurrency:
      typeof concurrencyLimit === 'number'
        ? formatCompactNumber(concurrencyLimit)
        : fallbackPlan.execConcurrency,
    execConcurrencyLimit:
      typeof concurrencyLimit === 'number'
        ? concurrencyLimit
        : fallbackPlan.execConcurrencyLimit,
    isCurrent: hasMappedBillingPlan,
    priceMonthly: formatCentsMonthly(plan?.amount) || fallbackPlan.priceMonthly,
  };

  return {
    currentPlan,
    currentPlanSku,
    plans: pricedPlans.map((candidate) =>
      hasMappedBillingPlan && candidate.sku === currentPlanSku
        ? currentPlan
        : { ...candidate, isCurrent: false },
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
  limit = 50,
  usage,
}: {
  functions?: WorkflowSummary[] | undefined;
  limit?: number;
  usage: WorkflowUsage[] | undefined;
}): TopFunctionRow[] {
  const summariesBySlug = new Map(
    functions?.map((fn) => [fn.slug, fn] as const) ?? [],
  );

  return (
    usage
      ?.filter((fn) => fn.dailyStarts.total > 0)
      ?.map((fn) => {
        const summary = summariesBySlug.get(fn.slug) ?? fn;
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
      .slice(0, limit) ?? []
  );
}
