import { useOrganization } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { Link } from '@inngest/components/Link';
import { createFileRoute } from '@tanstack/react-router';

import {
  HorizontalPlanCard,
  VerticalPlanCard,
} from '@/components/Billing/Plans/PlanCard';
import {
  HOBBY_PLAN_SLUG,
  PRO_PLAN_AMOUNT_CENTS,
  PRO_PLAN_SLUG,
} from '@/components/Billing/Plans/constants';
import { type Plan } from '@/components/Billing/Plans/utils';
import { currentPlan as getCurrentPlan } from '@/queries/server/billing';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/billing/plans/')({
  component: BillingPlansPage,
  loader: async () => {
    const { plan: currentPlan } = await getCurrentPlan();

    if (!currentPlan) throw new Error('Failed to fetch current plan');

    return {
      currentPlan,
    };
  },
});

function BillingPlansPage() {
  const { currentPlan } = Route.useLoaderData();
  const { isLoaded: orgLoaded, membership } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';

  const refetchCurrentPlan = async () => {
    return await getCurrentPlan();
  };

  //
  // Hard-coded plan information (mirrors pricing page definitions)
  const plans: Plan[] = [
    {
      id: 'n/a',
      slug: HOBBY_PLAN_SLUG,
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
      slug: PRO_PLAN_SLUG,
      name: 'Pro',
      amount: PRO_PLAN_AMOUNT_CENTS,
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
      {orgLoaded && !isAdmin && (
        <Alert severity="info" showIcon className="mb-8">
          Only organization admins can change your plan. Ask an admin to upgrade
          your account.
        </Alert>
      )}
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
        Cancel your plan by selecting the downgrade option. If you are having
        trouble downgrading, please{' '}
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
