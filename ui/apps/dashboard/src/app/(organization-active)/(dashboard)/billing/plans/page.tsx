import { NewLink } from '@inngest/components/Link/Link';

import { HorizontalPlanCard, VerticalPlanCard } from '@/components/Billing/Plans/PlanCard';
import { isEnterprisePlan } from '@/components/Billing/Plans/utils';
import { currentPlan as getCurrentPlan, plans as getPlans } from '@/components/Billing/data';
import type { BillingPlan } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';

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

  if (!currentPlan) throw new Error('Failed to fetch current plan');

  const refetchCurrentPlan = async () => {
    'use server';
    return await getCurrentPlan();
  };

  const isLegacyPlan =
    !isEnterprisePlan(currentPlan) && !plans.some((plan) => plan && plan.name === currentPlan.name);

  return (
    <>
      {isLegacyPlan && (
        <div className="mb-8">
          <HorizontalPlanCard
            plan={currentPlan}
            currentPlan={currentPlan}
            onPlanChange={refetchCurrentPlan}
          />
        </div>
      )}
      <p className="text-subtle mb-4">Available plans</p>
      <div className="mb-4 grid grid-cols-3 gap-4">
        {plans.map((plan) => {
          if (plan) {
            return (
              <VerticalPlanCard
                key={plan.id}
                plan={plan}
                currentPlan={currentPlan}
                onPlanChange={refetchCurrentPlan}
              />
            );
          }
        })}
      </div>
      <HorizontalPlanCard
        plan={ENTERPRISE_PLAN}
        currentPlan={currentPlan}
        onPlanChange={refetchCurrentPlan}
      />

      <div className="mt-4 text-center text-sm">
        Want to cancel your plan?{' '}
        <NewLink
          className="inline"
          target="_blank"
          size="small"
          href={pathCreator.support({ ref: 'app-billing-plans-footer' })}
        >
          Contact us
        </NewLink>
      </div>
    </>
  );
}
