import { NewButton } from '@inngest/components/Button';

import { currentPlan as getCurrentPlan } from '@/components/Billing/data';
import type { BillingPlan } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { PlanNames, isEnterprisePlan } from './utils';

export default async function UpgradeButton({ plan }: { plan: BillingPlan }) {
  const { plan: currentPlan } = await getCurrentPlan();
  if (!currentPlan) throw new Error('Failed to fetch current plan');

  const currentPlanName = currentPlan.name;
  const cardPlanName = plan.name;
  const currentPlanAmount = currentPlan.amount;
  const cardPlanAmount = plan.amount;

  const isEnterprise = isEnterprisePlan(currentPlan);

  const isActive =
    currentPlanName === cardPlanName || (cardPlanName === PlanNames.Enterprise && isEnterprise);

  const isLowerPlan = cardPlanAmount < currentPlanAmount;

  let buttonLabel = 'Upgrade';
  if (isActive) {
    buttonLabel = 'My Plan';
  } else if (isLowerPlan) {
    buttonLabel = 'Downgrade';
  }

  return (
    <div className="my-8">
      <NewButton className="w-full" label={buttonLabel} disabled={isActive || isEnterprise} />
      {isEnterprise && (
        <NewButton
          href={pathCreator.support({ ref: 'app-billing-plans-enterprise' })}
          label="Contact account manager"
          appearance="ghost"
          className="mt-1 w-full"
        />
      )}
    </div>
  );
}
