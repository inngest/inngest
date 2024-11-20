'use client';

import { NewButton } from '@inngest/components/Button';

import type { BillingPlan } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { PlanNames, isEnterprisePlan } from './utils';

export default async function UpgradeButton({
  plan,
  currentPlan,
}: {
  plan: BillingPlan;
  currentPlan: BillingPlan;
}) {
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
  } else if (cardPlanName === PlanNames.Enterprise) {
    buttonLabel = 'Get in touch';
  }

  const onClickChangePlan = () => {};

  return (
    <div className="my-8">
      <NewButton
        className="w-full"
        label={buttonLabel}
        disabled={isActive || isEnterprise}
        href={
          buttonLabel ? pathCreator.support({ ref: 'app-billing-plans-enterprise' }) : undefined
        }
        onClick={onClickChangePlan}
      />
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
