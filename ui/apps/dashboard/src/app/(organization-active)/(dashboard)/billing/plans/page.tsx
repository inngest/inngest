import { Link } from '@inngest/components/Link/Link';

import { HorizontalPlanCard, VerticalPlanCard } from '@/components/Billing/Plans/PlanCard';
import { isEnterprisePlan, isLegacyPlan, type Plan } from '@/components/Billing/Plans/utils';
import { currentPlan as getCurrentPlan, plans as getPlans } from '@/components/Billing/data';
import { pathCreator } from '@/utils/urls';

// This will move to the API as a custom plan at some point, for now we can hard code
const ENTERPRISE_PLAN: Plan = {
  id: 'n/a',
  name: 'Enterprise',
  amount: Infinity,
  billingPeriod: 'month',
  entitlements: {
    concurrency: { limit: 100000 },
    history: {
      limit: 90,
    },
    runCount: {
      limit: 100000000000,
    },
  },
};

export const dynamic = 'force-dynamic';

export default async function Page() {
  const plans = await getPlans();
  const { plan: currentPlan } = await getCurrentPlan();

  if (!currentPlan) throw new Error('Failed to fetch current plan');

  const refetchCurrentPlan = async () => {
    'use server';
    return await getCurrentPlan();
  };

  const isLegacy = isLegacyPlan(currentPlan);

  return (
    <>
      {isLegacy && (
        <div className="mb-8">
          <HorizontalPlanCard
            plan={currentPlan}
            currentPlan={currentPlan}
            onPlanChange={refetchCurrentPlan}
          />
        </div>
      )}
      <p className="text-subtle mb-4">Available plans</p>
      {isEnterprisePlan(currentPlan) && (
        <div className="mb-4">
          <HorizontalPlanCard
            plan={ENTERPRISE_PLAN}
            currentPlan={currentPlan}
            onPlanChange={refetchCurrentPlan}
          />
        </div>
      )}
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
      {!isEnterprisePlan(currentPlan) && (
        <HorizontalPlanCard
          plan={ENTERPRISE_PLAN}
          currentPlan={currentPlan}
          onPlanChange={refetchCurrentPlan}
        />
      )}
      <div className="mt-4 text-center text-sm">
        Want to cancel your plan?{' '}
        <Link
          className="inline"
          target="_blank"
          size="small"
          href={pathCreator.support({ ref: 'app-billing-plans-footer' })}
        >
          Contact us
        </Link>
      </div>
    </>
  );
}
