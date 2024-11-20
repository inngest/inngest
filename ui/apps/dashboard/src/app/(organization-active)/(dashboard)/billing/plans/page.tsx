// 'use client'

import { HorizontalPlanCard, VerticalPlanCard } from '@/components/Billing/Plans/PlanCard';
import { currentPlan as getCurrentPlan, plans as getPlans } from '@/components/Billing/data';
import type { BillingPlan } from '@/gql/graphql';

// This will move to the API as a custom plan at some point, for now we can hard code
const ENTERPRISE_PLAN: BillingPlan = {
  id: 'n/a',
  name: 'Enterprise',
  amount: Infinity,
  billingPeriod: 'month',
  features: {
    concurrency: 100000,
    log_retention: 90,
    runs: 100000000000,
  },
};

export default async function Page() {
  const plans = await getPlans();
  const { plan: currentPlan } = await getCurrentPlan();

  console.log(plans, currentPlan);

  return (
    <>
      <div className="mb-4 grid grid-cols-3 gap-4">
        {plans.map((plan) => {
          if (plan) {
            return <VerticalPlanCard plan={plan} />;
          }
        })}
      </div>
      <HorizontalPlanCard plan={ENTERPRISE_PLAN} />
    </>
  );
}
