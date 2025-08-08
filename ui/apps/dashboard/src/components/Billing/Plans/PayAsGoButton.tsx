'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';

import CheckoutModal, { type CheckoutItem } from '@/components/Billing/Plans/CheckoutModal';
import ConfirmPlanChangeModal from '@/components/Billing/Plans/ConfirmPlanChangeModal';
import { isHobbyFreePlan, isHobbyPaygPlan, type Plan } from './utils';

type ChangePlanArgs = {
  item: CheckoutItem;
  action: 'upgrade' | 'downgrade' | 'cancel';
};

export default function PayAsGoButton({
  plan,
  currentPlan,
  onPlanChange,
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

  let buttonLabel: string | undefined;

  if (isHobbyFreePlan(currentPlan)) {
    buttonLabel = 'Change to pay-as-you-go';
  } else {
    buttonLabel = 'Remove pay-as-you-go';
  }

  const onClickChangePlan = ({ item: { planSlug, name, amount }, action }: ChangePlanArgs) => {
    setCheckoutData({ items: [{ planSlug, name, quantity: 1, amount }], action });
  };

  const onChangePlanSuccess = () => {
    setCheckoutData(undefined);
    onPlanChange();
    router.refresh();
    toast.success(`Plan changed successfully`);
  };

  return (
    <>
      <Button
        appearance="outlined"
        label={buttonLabel}
        onClick={() => {
          onClickChangePlan({
            action: isHobbyPaygPlan(currentPlan) ? 'cancel' : 'upgrade',
            item: { planSlug: plan.slug, name: plan.name, quantity: 1, amount: plan.amount },
          });
        }}
      />
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
    </>
  );
}
