import { useState } from 'react';
import { useNavigate, useRouter } from '@tanstack/react-router';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';

import CheckoutModal, {
  type CheckoutItem,
} from '@/components/Billing/Plans/CheckoutModal';
import ConfirmPlanChangeModal from '@/components/Billing/Plans/ConfirmPlanChangeModal';
import { pathCreator } from '@/utils/urls';
import {
  PlanNames,
  isEnterprisePlan,
  isHobbyFreePlan,
  type Plan,
} from './utils';

type ChangePlanArgs = {
  item: CheckoutItem;
  action: 'upgrade' | 'downgrade' | 'cancel';
};

export default function UpgradeButton({
  plan,
  currentPlan,
  onPlanChange,
  label,
}: {
  plan: Plan;
  currentPlan: Plan;
  onPlanChange: () => void;
  label?: string;
}) {
  const router = useRouter();
  const [checkoutData, setCheckoutData] = useState<{
    action: 'upgrade' | 'downgrade' | 'cancel';
    items: CheckoutItem[];
  }>();
  const showCheckoutModal = checkoutData?.action === 'upgrade';
  const showChangePlanModal =
    checkoutData?.action === 'downgrade' || checkoutData?.action === 'cancel';

  const cardPlanName = plan.name;
  const currentPlanAmount = currentPlan.amount;
  const cardPlanAmount = plan.amount;

  const isEnterprise = isEnterprisePlan(currentPlan);
  const isEnterpriseCard = cardPlanName === PlanNames.Enterprise;
  const isFreeCard = cardPlanName === PlanNames.Free;

  const isActive =
    currentPlan.slug === plan.slug ||
    (cardPlanName === PlanNames.Enterprise && isEnterprise) ||
    false;

  const isLowerPlan = (() => {
    if (isEnterprise) {
      // If the current plan is Enterprise, all non-Enterprise plans are lower
      return !isEnterprisePlan(plan);
    }
    // For legacy plans, only Free Tier is a downgrade
    if (currentPlan.isLegacy && !currentPlan.isFree) {
      return isFreeCard || isHobbyFreePlan(plan);
    }
    // For non-enterprise plans, compare the amounts
    return cardPlanAmount < currentPlanAmount;
  })();

  let buttonLabel: string | undefined;
  if (isActive) {
    // Always override the label if the plan is active
    buttonLabel = 'My Plan';
  } else if (label) {
    buttonLabel = label;
  }

  if (!buttonLabel) {
    // If there still isn't a label then we need find a default

    if (isEnterpriseCard) {
      buttonLabel = 'Get in touch';
    } else if (isLowerPlan) {
      buttonLabel = 'Downgrade';
    } else {
      buttonLabel = 'Upgrade';
    }
  }

  const onClickChangePlan = ({
    item: { planSlug, name, amount },
    action,
  }: ChangePlanArgs) => {
    setCheckoutData({
      items: [{ planSlug, name, quantity: 1, amount }],
      action,
    });
  };

  const onChangePlanSuccess = () => {
    setCheckoutData(undefined);
    onPlanChange();
    //
    // Refresh the current route to reload data
    router.invalidate();
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
            action: isFreeCard
              ? 'cancel'
              : isLowerPlan
              ? 'downgrade'
              : 'upgrade',
            item: {
              planSlug: plan.slug,
              name: plan.name,
              quantity: 1,
              amount: plan.amount,
            },
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
