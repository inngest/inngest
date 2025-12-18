import { Alert } from '@inngest/components/Alert/NewAlert';
import { Button } from '@inngest/components/Button/NewButton';
import { Card } from '@inngest/components/Card/Card';
import { formatDayString } from '@inngest/components/utils/date';
import { createFileRoute } from '@tanstack/react-router';

import EntitlementListItem from '@/components/Billing/Addons/EntitlementListItem';
import BillingInformation from '@/components/Billing/BillingDetails/BillingInformation';
import PaymentMethod from '@/components/Billing/BillingDetails/PaymentMethod';
import { LimitBar, type Data } from '@/components/Billing/LimitBar';
import { isHobbyFreePlan, isHobbyPlan } from '@/components/Billing/Plans/utils';
import { ClientFeatureFlag } from '@/components/FeatureFlags/ClientFeatureFlag';
import {
  billingDetails as getBillingDetails,
  currentPlan as getCurrentPlan,
  entitlementUsage as getEntitlementUsage,
} from '@/queries/server/billing';
import { pathCreator } from '@/utils/urls';

const hasUsageMetrics = (obj: unknown): obj is { usage: number } => {
  return typeof obj === 'object' && obj !== null && 'usage' in obj;
};

export const Route = createFileRoute('/_authed/billing/')({
  component: BillingComponent,
  ssr: true,
  loader: async () => {
    const { addons, entitlements } = await getEntitlementUsage();

    const { plan: currentPlan, subscription: currentSubscription } =
      await getCurrentPlan();
    const billing = await getBillingDetails();

    if (!currentPlan) {
      throw new Error('Failed to fetch current plan');
    }

    let runs: Data | null = null;
    let steps: Data | null = null;
    let executions: Data | null = null;
    let isCurrentHobbyPlan = false;
    let legacyNoRunsPlan = false;

    //
    // usageMetricsCacheEnabled is checked inside getEntitlementUsage()
    // we can infer it's enabled if usage data is present
    const usageMetricsCacheEnabled = hasUsageMetrics(entitlements.stepCount);

    if (usageMetricsCacheEnabled) {
      isCurrentHobbyPlan = isHobbyPlan(currentPlan);
      legacyNoRunsPlan = entitlements.runCount.limit === null;

      const stepUsage = hasUsageMetrics(entitlements.stepCount)
        ? entitlements.stepCount.usage
        : 0;
      const stepLimit = entitlements.stepCount.limit;
      const runUsage = hasUsageMetrics(entitlements.runCount)
        ? entitlements.runCount.usage
        : 0;
      const runLimit = entitlements.runCount.limit;

      const executionsData = (entitlements as Record<string, unknown>)
        .executions;
      const executionUsage = hasUsageMetrics(executionsData)
        ? executionsData.usage
        : 0;
      const executionLimit =
        typeof executionsData === 'object' &&
        executionsData !== null &&
        'limit' in executionsData
          ? (executionsData.limit as number | null)
          : null;

      runs = {
        title: 'Runs',
        description: `${
          entitlements.runCount.overageAllowed
            ? 'Additional usage incurred at additional charge.'
            : ''
        }`,
        current: runUsage,
        limit: runLimit,
        overageAllowed: entitlements.runCount.overageAllowed,
        tooltipContent: 'A single durable function execution.',
      };

      const isExecutionBasedPlan =
        currentPlan.slug === 'pro-2025-08-08' ||
        currentPlan.slug === 'pro-2025-06-04';

      steps = {
        title: isExecutionBasedPlan ? 'Executions' : 'Steps',
        description: `${
          entitlements.runCount.overageAllowed && !legacyNoRunsPlan
            ? 'Additional usage incurred at additional charge. Additional runs include 5 steps per run.'
            : entitlements.runCount.overageAllowed
            ? 'Additional usage incurred at additional charge.'
            : ''
        }`,
        current: stepUsage,
        limit: stepLimit,
        overageAllowed: entitlements.stepCount.overageAllowed,
        tooltipContent: isExecutionBasedPlan
          ? 'Combined total of runs and steps executed.'
          : 'An individual step in durable functions.',
      };

      executions = {
        title: 'Executions',
        description: isCurrentHobbyPlan
          ? 'The maximum number of executions per month'
          : 'Additional usage billed at the start of the next billing cycle.',
        current: executionUsage,
        limit: executionLimit,
        overageAllowed:
          (entitlements as any).executions?.overageAllowed || false,
        tooltipContent: 'Combined total of runs and steps executed.',
      };
    }

    return {
      addons,
      entitlements,
      currentPlan,
      currentSubscription,
      billing,
      usageMetricsCacheEnabled,
      runs,
      steps,
      executions,
      isCurrentHobbyPlan,
      legacyNoRunsPlan,
    };
  },
});

const kbyteDisplayValue = (kibibytes: number): string => {
  if (kibibytes >= 1024) {
    return `${(kibibytes / 1024).toFixed(2)} MiB`;
  }
  return `${kibibytes} KiB`;
};

function BillingComponent() {
  const {
    addons,
    entitlements,
    currentPlan,
    currentSubscription,
    billing,
    usageMetricsCacheEnabled,
    runs,
    steps,
    executions,
    isCurrentHobbyPlan,
    legacyNoRunsPlan,
  } = Route.useLoaderData();

  const refetch = async () => {
    await getCurrentPlan();
    await getEntitlementUsage();
    await getBillingDetails();
  };

  const nextInvoiceDate = currentSubscription?.nextInvoiceDate
    ? formatDayString(new Date(currentSubscription.nextInvoiceDate))
    : undefined;

  const nextInvoiceAmount = currentPlan.amount
    ? `$${(currentPlan.amount / 100).toFixed(2)}`
    : 'Free';
  const overageAllowed =
    (entitlements.runCount.overageAllowed ||
      entitlements.stepCount.overageAllowed) &&
    !isHobbyFreePlan(currentPlan);

  const paymentMethod = billing.paymentMethods?.[0] || null;

  const advancedObservabilityAddon = {
    available: addons.advancedObservability.available,
    name: addons.advancedObservability.name,
    baseValue: addons.advancedObservability.purchased ? 1 : 0,
    maxValue: 1,
    quantityPer: 1,
    price: addons.advancedObservability.price,
    purchased: addons.advancedObservability.purchased,
  };

  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-2">
        {!overageAllowed && !isHobbyFreePlan(currentPlan) && (
          <Alert
            severity="info"
            className="flex items-center justify-between text-sm"
            link={
              <Button
                appearance="outlined"
                kind="secondary"
                label="Upgrade plan"
                href={pathCreator.billing({
                  tab: 'plans',
                  ref: 'app-billing-page-overview',
                })}
              />
            }
          >
            For usage beyond the limits of this plan, upgrade to a new plan.
          </Alert>
        )}
        {isHobbyFreePlan(currentPlan) && (
          <Alert
            severity="info"
            className="flex items-center justify-between text-sm"
            link={
              <Button
                appearance="outlined"
                kind="secondary"
                label="Upgrade plan"
                href={pathCreator.billing({
                  tab: 'plans',
                  ref: 'app-billing-page-overview',
                })}
              />
            }
          >
            For usage beyond the limits of this plan, upgrade to a new plan.
          </Alert>
        )}

        <Card.Content>
          <p className="text-muted mb-1">Your plan</p>
          <div className="flex items-center justify-between">
            <p className="text-basis text-xl">{currentPlan.name}</p>
            <Button
              appearance="ghost"
              label="Change plan"
              href={pathCreator.billing({
                tab: 'plans',
                ref: 'app-billing-page-overview',
              })}
            />
          </div>
          <div className="border-subtle mb-6 border" />
          {usageMetricsCacheEnabled && (
            <>
              {runs && !legacyNoRunsPlan && !isCurrentHobbyPlan && (
                <LimitBar data={runs} className="my-4" />
              )}
              {steps && !isCurrentHobbyPlan && (
                <LimitBar data={steps} className="mb-6" />
              )}
              {executions && isCurrentHobbyPlan && (
                <LimitBar data={executions} className="mb-6" />
              )}
            </>
          )}
          <EntitlementListItem
            planName={currentPlan.name}
            title="Concurrency"
            description="Maximum number of concurrently executing steps"
            tooltipContent="Functions actively sleeping and waiting for events are not counted"
            entitlement={{
              currentValue: entitlements.concurrency.limit,
              displayValue: `${entitlements.concurrency.limit} concurrent steps`,
            }}
            addon={addons.concurrency}
            onChange={refetch}
          />
          <EntitlementListItem
            planName={currentPlan.name}
            title="Users"
            description="Maximum number of users on the account"
            entitlement={{
              currentValue: entitlements.userCount.limit,
              displayValue: `${entitlements.userCount.usage} of ${entitlements.userCount.limit} maximum users`,
            }}
            addon={addons.userCount}
            onChange={refetch}
          />
          <ClientFeatureFlag flag="advanced-observability" defaultValue={false}>
            <EntitlementListItem
              increaseInHigherPlan={true}
              planName={currentPlan.name}
              title="Log retention"
              description="View and search function run traces and metrics"
              entitlement={{
                currentValue: addons.advancedObservability.purchased,
                displayValue: `${entitlements.history.limit} day${
                  entitlements.history.limit === 1 ? '' : 's'
                }`,
              }}
              addon={advancedObservabilityAddon}
              onChange={refetch}
            />
          </ClientFeatureFlag>
          <EntitlementListItem
            increaseInHigherPlan={true}
            planName={currentPlan.name}
            title="Metrics granularity"
            description="Granularity of exported metrics data points"
            entitlement={{
              currentValue: addons.advancedObservability.purchased,
              displayValue: `${
                entitlements.metricsExportGranularity.limit / 60
              } minute${
                entitlements.metricsExportGranularity.limit / 60 === 1
                  ? ''
                  : 's'
              }`,
            }}
            addon={advancedObservabilityAddon}
            onChange={refetch}
          />
          <EntitlementListItem
            increaseInHigherPlan={true}
            planName={currentPlan.name}
            title="Metrics freshness"
            description="How recent exported metrics data is"
            entitlement={{
              currentValue: addons.advancedObservability.purchased,
              displayValue: `${
                entitlements.metricsExportFreshness.limit / 60
              } minute${
                entitlements.metricsExportFreshness.limit / 60 === 1 ? '' : 's'
              }`,
            }}
            addon={advancedObservabilityAddon}
            onChange={refetch}
          />
          <ClientFeatureFlag
            flag="dedicated-slack-channel"
            defaultValue={false}
          >
            <EntitlementListItem
              increaseInHigherPlan={false}
              planName={currentPlan.name}
              title="Dedicated Slack Channel"
              description="Dedicated Slack channel for support"
              entitlement={{
                currentValue: entitlements.slackChannel.enabled,
                displayValue: entitlements.slackChannel.enabled
                  ? 'Enabled'
                  : 'Not enabled',
              }}
              addon={{
                ...addons.slackChannel,
                baseValue: 0,
                purchased: addons.slackChannel.purchaseCount > 0,
              }}
              onChange={refetch}
            />
          </ClientFeatureFlag>
          <ClientFeatureFlag flag="connect-workers" defaultValue={false}>
            <EntitlementListItem
              planName={currentPlan.name}
              title="Connect Workers"
              description="Maximum number of connect workers"
              entitlement={{
                currentValue: entitlements.connectWorkerConnections.limit,
                displayValue: `${entitlements.connectWorkerConnections.limit} connections`,
              }}
              addon={addons.connectWorkers}
              onChange={refetch}
            />
          </ClientFeatureFlag>
          <EntitlementListItem
            increaseInHigherPlan={false}
            planName={currentPlan.name}
            title="HIPAA"
            description="Sign BAAs for healthcare services"
            entitlement={{
              currentValue: entitlements.hipaa.enabled,
              displayValue: entitlements.hipaa.enabled
                ? 'Enabled'
                : 'Not enabled',
            }}
          />
          <EntitlementListItem
            planName={currentPlan.name}
            title="Event size"
            description="The maximum size for a single event"
            entitlement={{
              currentValue: entitlements.eventSize.limit,
              displayValue: kbyteDisplayValue(entitlements.eventSize.limit),
            }}
          />
          <div className="flex flex-col items-center gap-2 pt-6">
            <p className="text-muted text-xs">Custom needs?</p>
            <Button
              appearance="outlined"
              label="Chat with a product expert"
              href={pathCreator.support({ ref: 'app-billing-overview' })}
            />
          </div>
        </Card.Content>
      </Card>
      <div className="col-span-1">
        <Card className="mb-4">
          <Card.Content>
            <p className="text-muted mb-1">Next subscription payment</p>
            <p className="text-basis text-lg">
              {nextInvoiceAmount}
              {overageAllowed && (
                <span className="text-tertiary-moderate">*</span>
              )}
            </p>
            {nextInvoiceDate && (
              <>
                <p className="text-subtle mb-1 mt-4 text-xs font-medium">
                  Payment due date
                </p>
                <p className="text-basis text-sm">{nextInvoiceDate}</p>
              </>
            )}
            {overageAllowed && (
              <p className="text-subtle mt-4 text-xs italic">
                <span className="text-tertiary-moderate">*</span>Base plan cost.
                Additional usage calculated at the start of the next billing
                cycle.
              </p>
            )}
          </Card.Content>
        </Card>
        <BillingInformation
          billingEmail={billing.billingEmail}
          accountName={billing.name}
        />
        <PaymentMethod paymentMethod={paymentMethod} />
      </div>
    </div>
  );
}
