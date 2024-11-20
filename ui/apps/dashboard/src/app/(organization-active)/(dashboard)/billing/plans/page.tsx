import { NewLink } from '@inngest/components/Link/Link';

import { HorizontalPlanCard, VerticalPlanCard } from '@/components/Billing/Plans/PlanCard';
import { plans as getPlans } from '@/components/Billing/data';
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

  return (
    <>
      <p className="text-subtle mb-4">Available plans</p>
      <div className="mb-4 grid grid-cols-3 gap-4">
        {plans.map((plan) => {
          if (plan) {
            return <VerticalPlanCard key={plan.id} plan={plan} />;
          }
        })}
      </div>
      <HorizontalPlanCard plan={ENTERPRISE_PLAN} />

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
