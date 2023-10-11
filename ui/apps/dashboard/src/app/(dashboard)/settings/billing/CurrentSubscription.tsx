'use client';

import type { BillingPlan, BillingSubscription } from '@/gql/graphql';
import { day } from '@/utils/date';
import { FeatureRows, featureRows } from './BillingPlanOption';
import PaymentsButton from './PaymentsButton';
import PlanBadge from './PlanBadge';
import { type ExtendedBillingPlan } from './utils';

export default function CurrentSubscription({
  subscription,
  currentPlan,
  freePlan,
}: {
  subscription?: BillingSubscription;
  currentPlan?: BillingPlan;
  freePlan?: ExtendedBillingPlan;
}) {
  const nextInvoiceDate = subscription?.nextInvoiceDate
    ? day(subscription?.nextInvoiceDate)
    : undefined;
  const freeTierFeatureRows = featureRows.filter(({ showFreeTier }) => showFreeTier);

  return (
    <div className="rounded-lg border border-slate-300 bg-white">
      <div className="p-6">
        {nextInvoiceDate ? (
          <>
            <h2 className="mb-6 text-[1.375rem] font-semibold">Next Payment</h2>
            <p className="my-4 flex gap-4 text-xl">
              <span className="font-medium">
                ${currentPlan?.amount ? currentPlan?.amount / 100 : '0'}
              </span>
              <PlanBadge variant="primary">{currentPlan?.name}</PlanBadge>
            </p>
            <p className="my-4 text-sm text-slate-600">
              Next charge on <strong>{nextInvoiceDate}</strong>
            </p>
            <PaymentsButton />
          </>
        ) : (
          <>
            <h2 className="mb-6 text-[1.375rem] font-semibold">Your account is on the Free Tier</h2>
            <FeatureRows
              featureRows={freeTierFeatureRows}
              features={freePlan?.features || {}}
              isPremium={false}
            />
            <p className="mt-6">Select a plan to increase usage limits</p>
          </>
        )}
      </div>
    </div>
  );
}
