import { useMemo } from 'react';
import { useClient, useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
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

const zeroID = '00000000-0000-0000-0000-000000000000';
const functionCountPageSize = 1;
const topFunctionsUsagePageSize = 1000;
const topFunctionsLimit = 50;

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

export function useInfraDashboardData(timeRange: TimeRangeOption) {
  const env = useEnvironment();
  const client = useClient();
  const range = useMemo(() => getUtcMonthToDateRange(), [timeRange.id]);

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
      scope: MetricsScope.App,
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

  const data = useMemo(() => {
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
    const currentConcurrency = latestBucketMetricTotal(
      volume.data?.workspace.stepRunning.metrics.filter(
        ({ id }) => id !== zeroID,
      ),
    );
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
      billingNextInvoiceDate:
        currentPlan.data?.account.subscription?.nextInvoiceDate,
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
      stepRunning: latestMetricTotal(
        volume.data?.workspace.stepRunning.metrics,
      ),
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
      totalAccountConcurrency: volume.data?.accountConcurrency
        ? sumDataValues(volume.data.accountConcurrency.data)
        : 0,
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
    currentPlan.data?.account.subscription?.nextInvoiceDate,
    currentPlan.fetching,
    events.data?.environment.eventsV2.totalCount,
    functionUsage.data?.workspace.workflows.data,
    functions.data?.workspace.workflows.page,
    lookups.data?.envBySlug?.apps,
    lookups.data?.envBySlug?.workflows.data.length,
    volume.data?.accountConcurrency,
    volume.data?.workspace.backlog.metrics,
    volume.data?.workspace.runsThroughput.metrics,
    volume.data?.workspace.sdkThroughputEnded.metrics,
    volume.data?.workspace.sdkThroughputStarted.metrics,
    volume.data?.workspace.stepThroughput.metrics,
    volume.data?.workspace.stepRunning.metrics,
    volume.data?.workspace.workerPercentageUsed.metrics,
    volume.data?.workspace.workerTotalCapacity.metrics,
  ]);

  return {
    data,
    error:
      lookups.error ||
      functions.error ||
      functionUsage.error ||
      volume.error ||
      billableExecutions.error ||
      currentPlan.error ||
      availablePlans.error,
    eventsError: events.error,
    eventsFetching: events.fetching,
    fetching:
      lookups.fetching ||
      functions.fetching ||
      functionUsage.fetching ||
      volume.fetching ||
      billableExecutions.fetching ||
      currentPlan.fetching ||
      availablePlans.fetching,
    loading: {
      backlog: volume.fetching,
      billing: currentPlan.fetching || availablePlans.fetching,
      eventsReceived: events.fetching,
      executionsRan:
        billableExecutions.fetching ||
        (!billableExecutions.data?.usage && volume.fetching),
      executors: volume.fetching,
      queue: volume.fetching,
      topFunctions: functionUsage.fetching,
    },
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
