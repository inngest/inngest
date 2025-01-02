import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';

import AddOn from '@/components/Billing/Addons/AddonListItem';
import BillingInformation from '@/components/Billing/BillingDetails/BillingInformation';
import PaymentMethod from '@/components/Billing/BillingDetails/PaymentMethod';
import { LimitBar, type Data } from '@/components/Billing/LimitBar';
import {
  billingDetails as getBillingDetails,
  currentPlan as getCurrentPlan,
  entitlementUsage as getEntitlementUsage,
} from '@/components/Billing/data';
import { day } from '@/utils/date';
import { pathCreator } from '@/utils/urls';

export const dynamic = 'force-dynamic';

export default async function Page() {
  const entitlementUsage = await getEntitlementUsage();
  const plan = await getCurrentPlan();
  const billing = await getBillingDetails();

  const legacyNoRunsPlan = entitlementUsage.runCount.limit === null;
  const runs: Data = {
    title: 'Runs',
    description: `A single durable function execution. ${
      entitlementUsage.runCount.overageAllowed
        ? 'Additional usage incurred at additional charge.'
        : ''
    }`,
    current: entitlementUsage.runCount.usage || 0,
    limit: entitlementUsage.runCount.limit || null,
    overageAllowed: entitlementUsage.runCount.overageAllowed,
  };

  const steps: Data = {
    title: 'Steps',
    description: `An individual step in durable functions. ${
      entitlementUsage.runCount.overageAllowed && !legacyNoRunsPlan
        ? 'Additional usage incurred at additional charge. Additional runs include 5 steps per run.'
        : entitlementUsage.runCount.overageAllowed
        ? 'Additional usage incurred at additional charge.'
        : ''
    }`,
    current: entitlementUsage.stepCount.usage || 0,
    limit: entitlementUsage.stepCount.limit || null,
    overageAllowed: entitlementUsage.stepCount.overageAllowed,
  };

  const nextInvoiceDate = plan.subscription?.nextInvoiceDate
    ? day(plan.subscription.nextInvoiceDate)
    : undefined;

  const nextInvoiceAmount = plan.plan?.amount ? `$${plan.plan.amount / 100}` : 'Free';
  const overageAllowed =
    entitlementUsage.runCount.overageAllowed || entitlementUsage.stepCount.overageAllowed;

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
            <p className="text-basis text-xl">{plan.plan?.name}</p>
            <Button
              appearance="ghost"
              label="Change plan"
              href={pathCreator.billing({ tab: 'plans', ref: 'app-billing-page-overview' })}
            />
          </div>
          {!legacyNoRunsPlan && <LimitBar data={runs} className="my-4" />}
          <LimitBar data={steps} className="mb-6" />
          <div className="border-subtle mb-6 border" />
          <AddOn
            title="Event Size"
            value={
              entitlementUsage.eventSize.limit >= 1024
                ? `${(entitlementUsage.eventSize.limit / 1024).toFixed(2)} MB`
                : `${entitlementUsage.eventSize.limit} KB`
            }
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan}
            description="The maximum size for a single event"
          />
          <AddOn
            title="Concurrency"
            value={entitlementUsage.concurrency.limit}
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan}
            description="Maximum concurrently executing steps"
            tooltipContent="Functions actively sleeping and waiting for events are not counted"
          />
          <AddOn
            title="Log history"
            value={`${entitlementUsage.history.limit} day${
              entitlementUsage.history.limit === 1 ? '' : 's'
            }`}
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan}
            description="View and search function run traces and metrics"
          />
          <AddOn
            title="Dedicated execution capacity"
            canIncreaseLimitInCurrentPlan
            description="Dedicated infrastructure for the lowest latency and highest throughput"
          />
          <div className="flex flex-col items-center gap-2">
            <p>Custom needs?</p>
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
