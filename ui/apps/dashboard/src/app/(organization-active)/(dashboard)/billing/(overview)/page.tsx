import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { formatDayString } from '@inngest/components/utils/date';

import EntitlementListItem from '@/components/Billing/Addons/EntitlementListItem';
import BillingInformation from '@/components/Billing/BillingDetails/BillingInformation';
import PaymentMethod from '@/components/Billing/BillingDetails/PaymentMethod';
import { LimitBar, type Data } from '@/components/Billing/LimitBar';
import {
  billingDetails as getBillingDetails,
  currentPlan as getCurrentPlan,
  entitlementUsage as getEntitlementUsage,
} from '@/components/Billing/data';
import { pathCreator } from '@/utils/urls';

function kbyteDisplayValue(kibibytes: number): string {
  if (kibibytes >= 1024) {
    return `${(kibibytes / 1024).toFixed(2)} MiB`;
  }
  return `${kibibytes} KiB`;
}

function secondsToStr(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m == 0) {
    return `${s}s`;
  }
  if (s == 0) {
    return `${m}m`;
  }
  return `${m}m ${s}s`;
}

function metricExportDisplayValue(
  enabled: boolean,
  granularitySeconds: number,
  freshnessSeconds: number
): string | React.ReactNode {
  if (!enabled) {
    return 'Not enabled';
  }
  return (
    <>
      <span className="font-medium">Enabled</span>
      <br />
      <span className="text-muted">Granularity:</span>{' '}
      <span className="font-medium">{secondsToStr(granularitySeconds)}</span>
      <br />
      <span className="text-muted">Freshness:</span>{' '}
      <span className="font-medium">{secondsToStr(freshnessSeconds)}</span>
    </>
  );
}

export const dynamic = 'force-dynamic';

export default async function Page() {
  const { addons, entitlements } = await getEntitlementUsage();
  const { plan: currentPlan, subscription: currentSubscription } = await getCurrentPlan();
  const billing = await getBillingDetails();

  if (!currentPlan) {
    throw new Error('Failed to fetch current plan');
  }

  const refetch = async () => {
    'use server';
    await getCurrentPlan();
    await getEntitlementUsage();
    await getBillingDetails();
  };

  const legacyNoRunsPlan = entitlements.runCount.limit === null;
  const runs: Data = {
    title: 'Runs',
    description: `${
      entitlements.runCount.overageAllowed ? 'Additional usage incurred at additional charge.' : ''
    }`,
    current: entitlements.runCount.usage || 0,
    limit: entitlements.runCount.limit || null,
    overageAllowed: entitlements.runCount.overageAllowed,
    tooltipContent: 'A single durable function execution.',
  };

  const steps: Data = {
    title: 'Steps',
    description: `${
      entitlements.runCount.overageAllowed && !legacyNoRunsPlan
        ? 'Additional usage incurred at additional charge. Additional runs include 5 steps per run.'
        : entitlements.runCount.overageAllowed
        ? 'Additional usage incurred at additional charge.'
        : ''
    }`,
    current: entitlements.stepCount.usage || 0,
    limit: entitlements.stepCount.limit || null,
    overageAllowed: entitlements.stepCount.overageAllowed,
    tooltipContent: 'An individual step in durable functions.',
  };

  const nextInvoiceDate = currentSubscription?.nextInvoiceDate
    ? formatDayString(new Date(currentSubscription.nextInvoiceDate))
    : undefined;

  const nextInvoiceAmount = currentPlan.amount
    ? `$${(currentPlan.amount / 100).toFixed(2)}`
    : 'Free';
  const overageAllowed =
    entitlements.runCount.overageAllowed || entitlements.stepCount.overageAllowed;

  const paymentMethod = billing.paymentMethods?.[0] || null;

  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="col-span-2">
        {!overageAllowed && (
          <Alert
            severity="info"
            className="flex items-center justify-between text-sm"
            link={
              <Button
                appearance="outlined"
                kind="secondary"
                label="Upgrade plan"
                href={pathCreator.billing({ tab: 'plans', ref: 'app-billing-page-overview' })}
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
              href={pathCreator.billing({ tab: 'plans', ref: 'app-billing-page-overview' })}
            />
          </div>
          {!legacyNoRunsPlan && <LimitBar data={runs} className="my-4" />}
          <LimitBar data={steps} className="mb-6" />
          <div className="border-subtle mb-6 border" />
          <EntitlementListItem
            planName={currentPlan.name}
            title="Event size"
            description="The maximum size for a single event"
            entitlement={{
              currentValue: entitlements.eventSize.limit,
              displayValue: kbyteDisplayValue(entitlements.eventSize.limit),
            }}
          />
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
          <EntitlementListItem
            planName={currentPlan.name}
            title="Log history"
            description="View and search function run traces and metrics"
            entitlement={{
              currentValue: entitlements.history.limit,
              displayValue: `${entitlements.history.limit} day${
                entitlements.history.limit === 1 ? '' : 's'
              }`,
            }}
          />
          <EntitlementListItem
            increaseInHigherPlan={false}
            planName={currentPlan.name}
            title="HIPAA"
            description="Sign BAAs for healthcare services"
            entitlement={{
              currentValue: entitlements.hipaa.enabled,
              displayValue: entitlements.hipaa.enabled ? 'Enabled' : 'Not enabled',
            }}
          />
          <EntitlementListItem
            increaseInHigherPlan={false}
            planName={currentPlan.name}
            title="Dedicated execution capacity"
            description="Dedicated infrastructure for the lowest latency and highest throughput"
            entitlement={{
              currentValue: false,
              displayValue: 'Not enabled', // TODO: https://linear.app/inngest/issue/INN-4202/add-dedicated-capacity-addon
            }}
          />
          <EntitlementListItem
            increaseInHigherPlan={true}
            planName={currentPlan.name}
            title="Exportable metrics"
            description="Export key Inngest metrics into your own monitoring infrastructure"
            entitlement={{
              currentValue: entitlements.metricsExport.enabled,
              displayValue: metricExportDisplayValue(
                entitlements.metricsExport.enabled,
                entitlements.metricsExportGranularity.limit,
                entitlements.metricsExportFreshness.limit
              ),
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
              {overageAllowed && <span className="text-tertiary-moderate">*</span>}
            </p>
            {nextInvoiceDate && (
              <>
                <p className="text-subtle mb-1 mt-4 text-xs font-medium">Payment due date</p>
                <p className="text-basis text-sm">{nextInvoiceDate}</p>
              </>
            )}
            {overageAllowed && (
              <p className="text-subtle mt-4 text-xs italic">
                <span className="text-tertiary-moderate">*</span>Base plan cost. Additional usage
                calculated at the start of the next billing cycle.
              </p>
            )}
          </Card.Content>
        </Card>
        <BillingInformation billingEmail={billing.billingEmail} accountName={billing.name} />
        <PaymentMethod paymentMethod={paymentMethod} />
      </div>
    </div>
  );
}
