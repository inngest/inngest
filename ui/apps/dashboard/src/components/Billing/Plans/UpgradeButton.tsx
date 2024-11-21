'use client';

import { useState } from 'react';
import { NewButton } from '@inngest/components/Button';

import CheckoutModal, {
  type CheckoutItem,
} from '@/app/(organization-active)/(dashboard)/settings/billing/CheckoutModal';
import ConfirmPlanChangeModal from '@/app/(organization-active)/(dashboard)/settings/billing/ConfirmPlanChangeModal';
import type { BillingPlan } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { PlanNames, isEnterprisePlan } from './utils';

type ChangePlanArgs = {
  item: CheckoutItem;
  action: 'upgrade' | 'downgrade' | 'cancel';
};

export default function UpgradeButton({
  plan,
  currentPlan,
}: {
  plan: BillingPlan;
  currentPlan: BillingPlan;
}) {
  const [checkoutData, setCheckoutData] = useState<{
    action: 'upgrade' | 'downgrade' | 'cancel';
    items: CheckoutItem[];
  }>();
  const showCheckoutModal = checkoutData?.action === 'upgrade';
  const showChangePlanModal =
    checkoutData?.action === 'downgrade' || checkoutData?.action === 'cancel';

  const currentPlanName = currentPlan.name;
  const cardPlanName = plan.name;
  const currentPlanAmount = currentPlan.amount;
  const cardPlanAmount = plan.amount;

  const isEnterprise = isEnterprisePlan(currentPlan);
  const isEnterpriseCard = cardPlanName === PlanNames.Enterprise;
  const isFreeCard = cardPlanName === PlanNames.Free;

  const isActive =
    currentPlanName === cardPlanName || (cardPlanName === PlanNames.Enterprise && isEnterprise);

  const isLowerPlan = (() => {
    if (isEnterprise) {
      // If the current plan is Enterprise, all non-Enterprise plans are lower
      return !isEnterprisePlan(plan);
    }
    // For non-enterprise plans, compare the amounts
    return cardPlanAmount < currentPlanAmount;
  })();

  let buttonLabel = 'Upgrade';
  if (isActive) {
    buttonLabel = 'My Plan';
  } else if (isLowerPlan) {
    buttonLabel = 'Downgrade';
  } else if (isEnterpriseCard) {
    buttonLabel = 'Get in touch';
  }

  const onClickChangePlan = ({ item: { planID, name, amount }, action }: ChangePlanArgs) => {
    setCheckoutData({ items: [{ planID, name, quantity: 1, amount }], action });
  };

  const onChangePlanSuccess = () => {
    setCheckoutData(undefined);
    // OnUpdate(), to refresh plans page
  };

  return (
    <div className="my-8">
      <NewButton
        className="w-full"
        label={buttonLabel}
        disabled={isActive}
        href={
          isEnterpriseCard && !isActive
            ? pathCreator.support({ ref: 'app-billing-plans-enterprise' })
            : undefined
        }
        onClick={() => {
          if (isActive || isEnterpriseCard) return;
          onClickChangePlan({
            action: isFreeCard ? 'cancel' : isLowerPlan ? 'downgrade' : 'upgrade',
            item: { planID: plan.id, name: plan.name, quantity: 1, amount: plan.amount },
          });
        }}
      />
      {isEnterprise && (
        <NewButton
          href={pathCreator.support({ ref: 'app-billing-plans-enterprise' })}
          label="Contact account manager"
          appearance="ghost"
          className="mt-1 w-full"
        />
      )}
      {showCheckoutModal && (
        <CheckoutModal
          {...checkoutData}
          onSuccess={onChangePlanSuccess}
          onCancel={() => setCheckoutData(undefined)}
        />
      )}
      {showChangePlanModal && (
        <ConfirmPlanChangeModal
          {...checkoutData}
          onSuccess={onChangePlanSuccess}
          onCancel={() => setCheckoutData(undefined)}
        />
      )}
    </div>
  );
}
