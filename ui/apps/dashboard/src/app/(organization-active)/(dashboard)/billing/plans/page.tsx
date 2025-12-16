import { Link } from '@inngest/components/Link/Link';

import { HorizontalPlanCard, VerticalPlanCard } from '@/components/Billing/Plans/PlanCard';
import { type Plan } from '@/components/Billing/Plans/utils';
import { currentPlan as getCurrentPlan } from '@/components/Billing/data';
import { pathCreator } from '@/utils/urls';

export const dynamic = 'force-dynamic';

export default async function Page() {
  const { plan: currentPlan } = await getCurrentPlan();

  if (!currentPlan) throw new Error('Failed to fetch current plan');

  const refetchCurrentPlan = async () => {
    'use server';
    return await getCurrentPlan();
  };

  // Hard-coded plan information (mirrors pricing page definitions)
  const plans: Plan[] = [
    {
      id: 'n/a',
      slug: 'hobby-free-2025-08-08',
      name: 'Hobby',
      amount: 0,
      billingPeriod: 'month',
      entitlements: {
        concurrency: { limit: 5 },
        history: { limit: 1 }, // 24h
        runCount: { limit: 50_000 },
      },
      isLegacy: false,
      isFree: true,
    },
    {
      id: 'n/a',
      slug: 'pro-2025-08-08',
      name: 'Pro',
      amount: 7_500, // $75.00
      billingPeriod: 'month',
      entitlements: {
        concurrency: { limit: 100 },
        history: { limit: 7 },
        runCount: { limit: 1_000_000 },
      },
      isLegacy: false,
      isFree: false,
    },
    {
      id: 'n/a',
      slug: 'enterprise',
      name: 'Enterprise',
      amount: Infinity,
      billingPeriod: 'month',
      entitlements: {
        concurrency: { limit: 100 },
        history: { limit: 7 },
        runCount: { limit: 1_000_000 },
      },
      isLegacy: false,
      isFree: false,
    },
  ];

  return (
    <>
      {currentPlan.isLegacy && (
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
        {plans.map((plan) => (
          <VerticalPlanCard
            key={plan.id}
            plan={plan}
            currentPlan={currentPlan}
            onPlanChange={refetchCurrentPlan}
          />
        ))}
      </div>
      <div className="mt-4 text-center text-sm">
        Cancel your plan by selecting the downgrade option. If you are having trouble downgrading,
        please{' '}
        <Link
          className="inline"
          target="_blank"
          size="small"
          href={pathCreator.support({ ref: 'app-billing-plans-footer' })}
        >
          contact us
        </Link>
        .
      </div>
    </>
  );
}
