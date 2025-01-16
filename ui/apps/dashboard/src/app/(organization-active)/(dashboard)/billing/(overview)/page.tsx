import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';

import AddOn from '@/components/Billing/Addons/AddonListItem';
import BillingInformation from '@/components/Billing/BillingDetails/BillingInformation';
import PaymentMethod from '@/components/Billing/BillingDetails/PaymentMethod';
import { LimitBar, type Data } from '@/components/Billing/LimitBar';
import { PlanNames } from '@/components/Billing/Plans/utils';
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

  const legacyNoRunsPlan = entitlementUsage.runCount.limit === null;
  const runs: Data = {
    title: 'Runs',
    description: `${
      entitlementUsage.runCount.overageAllowed
        ? 'Additional usage incurred at additional charge.'
        : ''
    }`,
    current: entitlementUsage.runCount.usage || 0,
    limit: entitlementUsage.runCount.limit || null,
    overageAllowed: entitlementUsage.runCount.overageAllowed,
    tooltipContent: 'A single durable function execution.',
  };

  const steps: Data = {
    title: 'Steps',
    description: `${
      entitlementUsage.runCount.overageAllowed && !legacyNoRunsPlan
        ? 'Additional usage incurred at additional charge. Additional runs include 5 steps per run.'
        : entitlementUsage.runCount.overageAllowed
        ? 'Additional usage incurred at additional charge.'
        : ''
    }`,
    current: entitlementUsage.stepCount.usage || 0,
    limit: entitlementUsage.stepCount.limit || null,
    overageAllowed: entitlementUsage.stepCount.overageAllowed,
    tooltipContent: 'An individual step in durable functions.',
  };

  const nextInvoiceDate = currentSubscription?.nextInvoiceDate
    ? day(currentSubscription.nextInvoiceDate)
    : undefined;

  const nextInvoiceAmount = currentPlan.amount
    ? `$${(currentPlan.amount / 100).toFixed(2)}`
    : 'Free';
  const overageAllowed =
    entitlementUsage.runCount.overageAllowed || entitlementUsage.stepCount.overageAllowed;

  const paymentMethod = billing.paymentMethods?.[0] || null;

  const isProPlan = currentPlan.name === PlanNames.Pro;

  // TODO(cdzombak): various data is missing from the backend:
  //                 - canIncreaseLimitInCurrentPlan
  //                 - billing period
  //                 - maxValue
  //                 - addonName
  //                 - is override applied
  // TODO(cdzombak): self service must be unavailable if account override is applied
  // TODO(cdzombak): make most addonListItem inputs optional; refactor to make this flexibility cleaner
  // TODO(cdzombak): addonListItem must handle planLimit == null|undefined
  // TODO(cdzombak): hipaa addon
  // TODO(cdzombak): dedicated capacity addon

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
          <AddOn
            title="Event Size"
            value={entitlementUsage.eventSize.limit}
            displayValue={
              entitlementUsage.eventSize.limit >= 1024
                ? `${(entitlementUsage.eventSize.limit / 1024).toFixed(2)} MB`
                : `${entitlementUsage.eventSize.limit} KB`
            }
            planLimit={currentPlan.entitlements.eventSize.limit}
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan}
            description="The maximum size for a single event"
            selfServiceAvailable={false}
            maxValue={0} // unnecessary for no-self-service addons
            quantityPer={0} // unnecessary for no-self-service addons
            addonName={''} // unnecessary for no-self-service addons
          />
          <AddOn
            title="Concurrency"
            value={entitlementUsage.concurrency.limit}
            displayValue={`${entitlementUsage.concurrency.limit} concurrent steps`}
            canIncreaseLimitInCurrentPlan={
              entitlementUsage.isCustomPlan || currentPlan.addons.concurrency.available
            }
            planLimit={currentPlan.entitlements.concurrency.limit}
            maxValue={1000} // TODO(cdzombak): where should this come from?
            quantityPer={currentPlan.addons.concurrency.quantityPer}
            description="Maximum number of concurrently executing steps"
            tooltipContent="Functions actively sleeping and waiting for events are not counted"
            selfServiceAvailable={!!currentPlan.addons.concurrency.price}
            price={currentPlan.addons.concurrency.price || undefined}
            addonName={'concurrency'}
            onChange={refetch}
          />
          <AddOn
            title="Users"
            value={entitlementUsage.userCount.limit || 0}
            displayValue={`${entitlementUsage.userCount.usage} of ${entitlementUsage.userCount.limit} maximum users`}
            canIncreaseLimitInCurrentPlan={currentPlan.addons.userCount.available}
            description="Maximum number of users on the account"
            planLimit={currentPlan.entitlements.userCount.limit || -1}
            maxValue={1000} // TODO(cdzombak): where should this come from?
            quantityPer={currentPlan.addons.userCount.quantityPer}
            selfServiceAvailable={
              !!currentPlan.addons.userCount.price && entitlementUsage.userCount.limit !== null
            }
            price={currentPlan.addons.userCount.price || undefined}
            addonName={'!!!user_count'}
            onChange={refetch}
          />
          <AddOn
            title="Log history"
            value={entitlementUsage.history.limit}
            displayValue={`${entitlementUsage.history.limit} day${
              entitlementUsage.history.limit === 1 ? '' : 's'
            }`}
            planLimit={currentPlan.entitlements.history.limit}
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan}
            description="View and search function run traces and metrics"
            selfServiceAvailable={false}
            maxValue={366} // unnecessary for no-self-service addons
            quantityPer={7} // unnecessary for no-self-service addons
            addonName={''} // unnecessary for no-self-service addons
          />
          <AddOn
            title="HIPAA"
            value={entitlementUsage.hipaa.enabled}
            displayValue={entitlementUsage.hipaa.enabled ? 'Enabled' : 'Not enabled'}
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan || isProPlan}
            description="Sign BAAs for healthcare services"
            planLimit={1} // TODO(cdzombak): nonsense for boolean
            maxValue={1} // TODO(cdzombak): nonsense for boolean
            quantityPer={1} // TODO(cdzombak): nonsense for boolean
            selfServiceAvailable={false} // TODO(cdzombak): should be true eventually
            addonName={'hipaa'}
          />
          <AddOn
            title="Dedicated execution capacity"
            canIncreaseLimitInCurrentPlan={entitlementUsage.isCustomPlan}
            description="Dedicated infrastructure for the lowest latency and highest throughput"
            selfServiceAvailable={false}
            value={0} // TODO(cdzombak): need this from the backend
            displayValue={'Not enabled'}
            maxValue={1000} // TODO(cdzombak): where should this come from?
            planLimit={0} // TODO(cdzombak): where should this come from?
            quantityPer={250} // TODO(cdzombak): where should this come from?
            price={500}
            addonName={''}
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
