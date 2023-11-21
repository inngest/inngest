'use client';

import type { BillingPlan, BillingSubscription } from '@/gql/graphql';
import { day } from '@/utils/date';
import { FeatureRows, featureRows } from './BillingPlanOption';
import PaymentsButton from './PaymentsButton';
import PlanBadge from './PlanBadge';
import { isTrialPlan, type ExtendedBillingPlan } from './utils';

export default function CurrentSubscription({
  subscription,
  currentPlan,
  isCurrentPlanEnterprise,
  freePlan,
}: {
  subscription?: BillingSubscription;
  currentPlan?: BillingPlan;
  isCurrentPlanEnterprise: boolean;
  freePlan?: ExtendedBillingPlan;
}) {
  const isOnPaidPlan = isCurrentPlanEnterprise || currentPlan?.amount !== 0;
  const isTrial = currentPlan ? isTrialPlan(currentPlan) : false;
  const nextInvoiceDate = subscription?.nextInvoiceDate
    ? day(subscription?.nextInvoiceDate)
    : undefined;
  const freeTierFeatureRows = featureRows.filter(({ showFreeTier }) => showFreeTier);

  return (
    <div className="rounded-lg border border-slate-300 bg-white">
      <div className="p-6">
        {isOnPaidPlan ? (
          <>
            <h2 className="mb-6 text-[1.375rem] font-semibold">
              {isTrial ? 'Your account is on a trial plan' : 'Next Payment'}
            </h2>
            <p className="my-4 flex flex-wrap gap-4 text-xl">
              {!isTrial && (
                <span className="font-medium">
                  ${currentPlan?.amount ? (currentPlan?.amount / 100).toLocaleString() : '0'}
                </span>
              )}
              <PlanBadge variant="primary" className="whitespace-nowrap">
                {currentPlan?.name}
              </PlanBadge>
            </p>
            {!!nextInvoiceDate && (
              <>
                <p className="my-4 text-sm text-slate-600">
                  Next charge on <strong>{nextInvoiceDate}</strong>
                </p>
                <PaymentsButton />
              </>
            )}
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
