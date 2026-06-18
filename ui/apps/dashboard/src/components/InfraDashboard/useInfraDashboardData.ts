import { useEffect, useMemo, useState } from 'react';
import { useClient, useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import {
  latestMetricDataValue,
  sumScopedMetricData,
} from '@/components/Metrics/metricAggregation';
import { graphql } from '@/gql';
import {
  GetBillableExecutionsDocument,
  GetCurrentPlanDocument,
  GetFunctionsDocument,
  GetFunctionsUsageDocument,
  GetPlansDocument,
  MetricsLookupsDocument,
  MetricsScope,
  VolumeMetricsDocument,
  type GetCurrentPlanQuery,
} from '@/gql/graphql';

import { INFRA_DASHBOARD_PLACEHOLDERS } from './placeholderData';
import {
  buildTopFunctionRows,
  getUtcMonthToDateRange,
  latestBucketMetricTotal,
  latestMetricTotal,
  mergeBillingPlanIntoInfraPlans,
  isEnterprisePlanName,
  pickCheapestEnabledProPlanAmount,
  pickInfraConcurrencyAddon,
  sumDataValues,
  sumMetricValues,
  sumTimeSeriesValues,
} from './utils';

export type TimeRangeOption = {
  id: 'month';
  name: string;
};

export const TIME_RANGE_OPTIONS: TimeRangeOption[] = [
  { id: 'month', name: 'This month' },
];

const cacheTTL = 60 * 60 * 1000;
const cacheVersion = 3;
const functionCountPageSize = 1;
const topFunctionsUsagePageSize = 1000;
const topFunctionsLimit = 50;

type InfraDashboardData = {
  accountConcurrencyLimit: number;
  appsCount: number;
  backlogDepth: number;
  billingActionsReady: boolean;
  billingPlanReady: boolean;
  concurrencyAddon: ReturnType<typeof pickInfraConcurrencyAddon>;
  currentBillingPlan: GetCurrentPlanQuery['account']['plan'] | undefined;
  currentInfraPlan: ReturnType<
    typeof mergeBillingPlanIntoInfraPlans
  >['currentPlan'];
  currentInfraPlanSku: ReturnType<
    typeof mergeBillingPlanIntoInfraPlans
  >['currentPlanSku'];
  currentConcurrency: number;
  eventsReceived: number;
  executionsRan: number;
  functionsCount: number;
  functionsRan: number;
  infraPlans: ReturnType<typeof mergeBillingPlanIntoInfraPlans>['plans'];
  hasPaymentMethod: boolean;
  isEnterprisePlan: boolean;
  planName: string;
  placeholders: typeof INFRA_DASHBOARD_PLACEHOLDERS;
  proPlanAmountCents: number | null;
  sdkRequests: number;
  stepRunning: number;
  topFunctions: ReturnType<typeof buildTopFunctionRows>;
  workerCapacity: number;
  workerPercentUsed: number | null;
  totalAccountConcurrency: number;
};

type InfraDashboardCacheEntry = {
  data: InfraDashboardData;
  savedAt: number;
  version: number;
};

const InfraDashboardEventsCountDocument = graphql(`
  query InfraDashboardEventsCount(
    $envID: ID!
    $startTime: Time!
    $endTime: Time
  ) {
    environment: workspace(id: $envID) {
      eventsV2(
        filter: {
          from: $startTime
          until: $endTime
          includeInternalEvents: false
        }
      ) {
        totalCount
      }
    }
  }
`);

function cacheKey({
  envID,
  month,
  year,
}: {
  envID: string;
  month: number;
  year: number;
}) {
  return `inngest:infra-dashboard:v${cacheVersion}:${envID}:${year}-${month}`;
}

function readCache(key: string): InfraDashboardCacheEntry | null {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) {
      return null;
    }

    const cached = JSON.parse(raw) as InfraDashboardCacheEntry;
    if (
      cached.version !== cacheVersion ||
      typeof cached.savedAt !== 'number' ||
      Date.now() - cached.savedAt > cacheTTL
    ) {
      window.localStorage.removeItem(key);
      return null;
    }

    return cached;
  } catch {
    window.localStorage.removeItem(key);
    return null;
  }
}

function writeCache(
  key: string,
  data: InfraDashboardData,
): InfraDashboardCacheEntry | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const entry: InfraDashboardCacheEntry = {
    data,
    savedAt: Date.now(),
    version: cacheVersion,
  };

  try {
    window.localStorage.setItem(key, JSON.stringify(entry));
    return entry;
  } catch {
    return null;
  }
}

export function useInfraDashboardData(timeRange: TimeRangeOption) {
  const env = useEnvironment();
  const client = useClient();
  const range = useMemo(() => getUtcMonthToDateRange(), [timeRange.id]);
  const key = useMemo(
    () => cacheKey({ envID: env.id, month: range.month, year: range.year }),
    [env.id, range.month, range.year],
  );
  const [cached, setCached] = useState<InfraDashboardCacheEntry | null>(null);

  useEffect(() => {
    setCached(readCache(key));
  }, [key]);

  const [lookups] = useQuery({
    query: MetricsLookupsDocument,
    variables: { envSlug: env.slug, page: 1, pageSize: 1000 },
  });

  const [functions] = useQuery({
    query: GetFunctionsDocument,
    variables: {
      archived: false,
      environmentID: env.id,
      page: 1,
      pageSize: functionCountPageSize,
      search: null,
    },
  });

  const [functionUsage] = useQuery({
    query: GetFunctionsUsageDocument,
    variables: {
      archived: false,
      environmentID: env.id,
      page: 1,
      pageSize: topFunctionsUsagePageSize,
    },
  });

  const [events] = useQuery({
    query: InfraDashboardEventsCountDocument,
    variables: {
      endTime: range.until.toISOString(),
      envID: env.id,
      startTime: range.from.toISOString(),
    },
  });

  const [volume] = useQuery({
    query: VolumeMetricsDocument,
    variables: {
      appIDs: [],
      from: range.from.toISOString(),
      functionIDs: [],
      scope: MetricsScope.Fn,
      until: range.until.toISOString(),
      workspaceId: env.id,
    },
  });

  const [billableExecutions] = useQuery({
    query: GetBillableExecutionsDocument,
    variables: {
      month: range.month,
      year: range.year,
    },
  });

  const [currentPlan, refetchCurrentPlan] = useQuery({
    query: GetCurrentPlanDocument,
  });
  const [availablePlans, refetchAvailablePlans] = useQuery({
    query: GetPlansDocument,
  });

  const liveData = useMemo<InfraDashboardData>(() => {
    const activeApps =
      lookups.data?.envBySlug?.apps.filter((app) => !app.isArchived) ?? [];
    const workflowPage = functions.data?.workspace.workflows.page;
    const usageRows = functionUsage.data?.workspace.workflows.data;
    const functionsRan =
      usageRows?.reduce((total, fn) => total + fn.dailyStarts.total, 0) ?? 0;
    const runsEnded = sumMetricValues(
      volume.data?.workspace.runsThroughput.metrics,
    );
    const stepsRan = sumMetricValues(
      volume.data?.workspace.stepThroughput.metrics,
    );
    const hasBillableExecutions = Boolean(billableExecutions.data?.usage);
    const billableExecutionsRan = sumTimeSeriesValues(
      billableExecutions.data?.usage,
    );
    const backlogDepth = latestBucketMetricTotal(
      volume.data?.workspace.backlog.metrics,
    );
    const accountConcurrency = sumScopedMetricData(
      volume.data?.workspace.stepRunning.metrics,
    );
    const currentConcurrency = latestMetricDataValue(accountConcurrency);
    const proPlanAmountCents = pickCheapestEnabledProPlanAmount(
      availablePlans.data?.plans,
    );
    const billingPlan = mergeBillingPlanIntoInfraPlans({
      accountEntitlements: currentPlan.data?.account.entitlements,
      defaultSku: INFRA_DASHBOARD_PLACEHOLDERS.defaultPlanSku,
      plan: currentPlan.data?.account.plan,
      plans: INFRA_DASHBOARD_PLACEHOLDERS.infraPlans,
      proPlanAmountCents,
    });
    const billingPlanReady = Boolean(
      !currentPlan.fetching &&
        currentPlan.data?.account.plan &&
        currentPlan.data?.account.entitlements,
    );

    return {
      accountConcurrencyLimit: billingPlan.currentPlan.execConcurrencyLimit,
      appsCount: activeApps.length,
      backlogDepth,
      billingActionsReady: Boolean(
        !currentPlan.fetching && currentPlan.data?.account.plan,
      ),
      billingPlanReady,
      concurrencyAddon: pickInfraConcurrencyAddon({
        accountAddon: currentPlan.data?.account.addons?.concurrency,
        planAddon: currentPlan.data?.account.plan?.addons.concurrency,
      }),
      currentBillingPlan: currentPlan.data?.account.plan,
      currentInfraPlan: billingPlan.currentPlan,
      currentInfraPlanSku: billingPlan.currentPlanSku,
      currentConcurrency,
      eventsReceived: events.data?.environment.eventsV2.totalCount ?? 0,
      executionsRan: hasBillableExecutions
        ? billableExecutionsRan
        : runsEnded + stepsRan,
      functionsCount:
        workflowPage?.totalItems ??
        lookups.data?.envBySlug?.workflows.data.length ??
        0,
      functionsRan: functionsRan || runsEnded,
      infraPlans: billingPlan.plans,
      hasPaymentMethod: Boolean(
        currentPlan.data?.account.paymentMethods?.length,
      ),
      isEnterprisePlan: isEnterprisePlanName(
        currentPlan.data?.account.plan?.name,
      ),
      planName: currentPlan.data?.account.plan?.name ?? 'Plan',
      placeholders: INFRA_DASHBOARD_PLACEHOLDERS,
      proPlanAmountCents,
      sdkRequests:
        sumMetricValues(volume.data?.workspace.sdkThroughputStarted.metrics) ||
        sumMetricValues(volume.data?.workspace.sdkThroughputEnded.metrics),
      stepRunning: currentConcurrency,
      topFunctions: buildTopFunctionRows({
        limit: topFunctionsLimit,
        usage: usageRows,
      }),
      workerCapacity: latestMetricTotal(
        volume.data?.workspace.workerTotalCapacity.metrics,
      ),
      workerPercentUsed:
        volume.data?.workspace.workerPercentageUsed.metrics.at(0)?.data.at(-1)
          ?.value ?? null,
      totalAccountConcurrency: sumDataValues(accountConcurrency),
    };
  }, [
    availablePlans.data?.plans,
    billableExecutions.data?.usage,
    currentPlan.data?.account.addons?.concurrency,
    currentPlan.data?.account.entitlements,
    currentPlan.data?.account.paymentMethods,
    currentPlan.data?.account.plan,
    currentPlan.data?.account.plan?.addons.concurrency,
    currentPlan.data?.account.plan?.name,
    currentPlan.fetching,
    events.data?.environment.eventsV2.totalCount,
    functionUsage.data?.workspace.workflows.data,
    functions.data?.workspace.workflows.page,
    lookups.data?.envBySlug?.apps,
    lookups.data?.envBySlug?.workflows.data.length,
    volume.data?.workspace.backlog.metrics,
    volume.data?.workspace.runsThroughput.metrics,
    volume.data?.workspace.sdkThroughputEnded.metrics,
    volume.data?.workspace.sdkThroughputStarted.metrics,
    volume.data?.workspace.stepThroughput.metrics,
    volume.data?.workspace.stepRunning.metrics,
    volume.data?.workspace.workerPercentageUsed.metrics,
    volume.data?.workspace.workerTotalCapacity.metrics,
  ]);

  const liveDataReady = Boolean(
    lookups.data &&
      functions.data &&
      functionUsage.data &&
      events.data &&
      volume.data &&
      billableExecutions.data &&
      currentPlan.data &&
      availablePlans.data &&
      !lookups.fetching &&
      !functions.fetching &&
      !functionUsage.fetching &&
      !events.fetching &&
      !volume.fetching &&
      !billableExecutions.fetching &&
      !currentPlan.fetching &&
      !availablePlans.fetching,
  );
  const liveError =
    lookups.error ||
    functions.error ||
    functionUsage.error ||
    events.error ||
    volume.error ||
    billableExecutions.error ||
    currentPlan.error ||
    availablePlans.error;
  const isUsingCachedData = Boolean(cached && !liveDataReady);
  const data = isUsingCachedData && cached ? cached.data : liveData;
  const fetching =
    !isUsingCachedData &&
    (lookups.fetching ||
      functions.fetching ||
      functionUsage.fetching ||
      volume.fetching ||
      billableExecutions.fetching ||
      currentPlan.fetching ||
      availablePlans.fetching);
  const loading = isUsingCachedData
    ? {
        backlog: false,
        billing: false,
        eventsReceived: false,
        executionsRan: false,
        executors: false,
        queue: false,
        topFunctions: false,
      }
    : {
        backlog: volume.fetching,
        billing: currentPlan.fetching || availablePlans.fetching,
        eventsReceived: events.fetching,
        executionsRan:
          billableExecutions.fetching ||
          (!billableExecutions.data?.usage && volume.fetching),
        executors: volume.fetching,
        queue: volume.fetching,
        topFunctions: functionUsage.fetching,
      };

  useEffect(() => {
    if (!liveDataReady || liveError) {
      return;
    }

    const entry = writeCache(key, liveData);
    if (entry) {
      setCached(entry);
    }
  }, [key, liveData, liveDataReady, liveError]);

  return {
    cacheStatus: {
      cachedAt: cached?.savedAt,
      isUsingCachedData,
      ttlMs: cacheTTL,
    },
    data,
    error: !isUsingCachedData && liveError ? liveError : undefined,
    eventsError: !isUsingCachedData ? events.error : undefined,
    eventsFetching: !isUsingCachedData && events.fetching,
    fetching,
    loading,
    range,
    refetchBillingData: async () => {
      await client
        .query(GetCurrentPlanDocument, {}, { requestPolicy: 'network-only' })
        .toPromise();
      await client
        .query(GetPlansDocument, {}, { requestPolicy: 'network-only' })
        .toPromise();
      refetchCurrentPlan({ requestPolicy: 'network-only' });
      refetchAvailablePlans({ requestPolicy: 'network-only' });
    },
  };
}
