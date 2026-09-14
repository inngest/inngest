import { useOrganization } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { Link } from '@inngest/components/Link';
import { createFileRoute } from '@tanstack/react-router';

import {
  HorizontalPlanCard,
  VerticalPlanCard,
} from '@/components/Billing/Plans/PlanCard';
import {
  pickSelfServePlans,
  type Plan,
} from '@/components/Billing/Plans/utils';
import {
  currentPlan as getCurrentPlan,
  plans as getPlans,
} from '@/queries/server/billing';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/billing/plans/')({
  component: BillingPlansPage,
  loader: async () => {
    const [{ plan: currentPlan }, availablePlans] = await Promise.all([
      getCurrentPlan(),
      getPlans(),
    ]);

    if (!currentPlan) throw new Error('Failed to fetch current plan');
    const selfServePlans = pickSelfServePlans(availablePlans);
    const availableSelfServePlans = [
      selfServePlans.hobby,
      selfServePlans.pro,
    ].filter((plan) => plan !== null);
    if (availableSelfServePlans.length === 0) {
      throw new Error('Failed to fetch available plans');
    }

    return {
      availableSelfServePlans,
      currentPlan,
    };
  },
});

function BillingPlansPage() {
  const { availableSelfServePlans, currentPlan } = Route.useLoaderData();
  const { isLoaded: orgLoaded, membership } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';

  const refetchCurrentPlan = async () => {
    return await getCurrentPlan();
  };

  const plans: Plan[] = [
    ...availableSelfServePlans,
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
