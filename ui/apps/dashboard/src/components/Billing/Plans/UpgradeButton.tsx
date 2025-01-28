'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';

import CheckoutModal, { type CheckoutItem } from '@/components/Billing/Plans/CheckoutModal';
import ConfirmPlanChangeModal from '@/components/Billing/Plans/ConfirmPlanChangeModal';
import { pathCreator } from '@/utils/urls';
import { PlanNames, isEnterprisePlan, isLegacyPlan, type Plan } from './utils';

type ChangePlanArgs = {
  item: CheckoutItem;
  action: 'upgrade' | 'downgrade' | 'cancel';
};

export default function UpgradeButton({
  plan,
  currentPlan,
  onPlanChange,
}: {
  plan: Plan;
  currentPlan: Plan;
  onPlanChange: () => void;
}) {
  const router = useRouter();
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

  const isLegacy = isLegacyPlan(currentPlan);

  const isActive =
    currentPlanName === cardPlanName || (cardPlanName === PlanNames.Enterprise && isEnterprise);

  const isLowerPlan = (() => {
    if (isEnterprise) {
      // If the current plan is Enterprise, all non-Enterprise plans are lower
      return !isEnterprisePlan(plan);
    }
    // For legacy plans, only Free Tier is a downgrade
    if (isLegacy) {
      return isFreeCard;
    }
    // For non-enterprise plans, compare the amounts
    return cardPlanAmount < currentPlanAmount;
  })();

  const buttonLabel = isActive
    ? 'My Plan'
    : isEnterpriseCard
    ? 'Get in touch'
    : isLowerPlan
    ? 'Downgrade'
    : 'Upgrade';

  const onClickChangePlan = ({ item: { planID, name, amount }, action }: ChangePlanArgs) => {
    setCheckoutData({ items: [{ planID, name, quantity: 1, amount }], action });
  };

  const onChangePlanSuccess = () => {
    setCheckoutData(undefined);
    onPlanChange();
    router.refresh();
    toast.success(`Plan changed successfully`);
  };

  return (
    <div className="my-8">
      <Button
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
      {isEnterprise && isEnterpriseCard && (
        <Button
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
