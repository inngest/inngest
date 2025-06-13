import { Link } from '@inngest/components/Link/Link';

import { HorizontalPlanCard, VerticalPlanCard } from '@/components/Billing/Plans/PlanCard';
import { isEnterprisePlan, isLegacyPlan, type Plan } from '@/components/Billing/Plans/utils';
import { currentPlan as getCurrentPlan } from '@/components/Billing/data';
import { pathCreator } from '@/utils/urls';

// This will move to the API as a custom plan at some point, for now we can hard code
const ENTERPRISE_PLAN: Plan = {
  id: 'n/a',
  name: 'Enterprise',
  amount: Infinity,
  billingPeriod: 'month',
  entitlements: {
    concurrency: { limit: 500 },
    history: {
      limit: 30, // days
    },
    runCount: {
      limit: 100000000000,
    },
  },
};

export const dynamic = 'force-dynamic';

export default async function Page() {
  const { plan: currentPlan } = await getCurrentPlan();

  if (!currentPlan) throw new Error('Failed to fetch current plan');

  const refetchCurrentPlan = async () => {
    'use server';
    return await getCurrentPlan();
  };

  const isLegacy = isLegacyPlan(currentPlan);

  // Hard-coded plan information (mirrors pricing page definitions)
  const plans: Plan[] = [
    {
      id: 'hobby',
      name: 'Hobby',
      amount: 0,
      billingPeriod: 'month',
      entitlements: {
        concurrency: { limit: 25 },
        history: { limit: 1 }, // 24h
        runCount: { limit: 100_000 },
      },
    },
    {
      id: 'hobby-payg',
      name: 'Hobby (Pay as you go)',
      amount: 0,
      billingPeriod: 'month',
      entitlements: {
        concurrency: { limit: 25 },
        history: { limit: 1 }, // 24h
        runCount: { limit: 1_000_000 },
      },
    },
    {
      id: 'pro',
      name: 'Pro',
      amount: 7_500, // $75.00
      billingPeriod: 'month',
      entitlements: {
        concurrency: { limit: 100 },
        history: { limit: 7 },
        runCount: { limit: 1_000_000 },
      },
    },
  ];

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
        {plans.map((plan) => (
          <VerticalPlanCard
            key={plan.id}
            plan={plan}
            currentPlan={currentPlan}
            onPlanChange={refetchCurrentPlan}
          />
        ))}
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
