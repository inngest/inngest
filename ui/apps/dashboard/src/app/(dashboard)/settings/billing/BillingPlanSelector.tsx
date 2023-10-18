'use client';

import { useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import cn from '@/utils/cn';
import { WEBSITE_CONTACT_URL, WEBSITE_PRICING_URL } from '@/utils/urls';
import BillingPlanOption, { type ChangePlanArgs } from './BillingPlanOption';
import CheckoutModal, { type CheckoutItem } from './CheckoutModal';
import ConfirmPlanChangeModal from './ConfirmPlanChangeModal';
import { type ExtendedBillingPlan } from './utils';

export default function BillingPlanSelector({
  plans,
  isCurrentPlanEnterprise,
}: {
  plans: (ExtendedBillingPlan | null)[];
  isCurrentPlanEnterprise: boolean;
}) {
  const router = useRouter();
  const [checkoutData, setCheckoutData] = useState<{
    action: 'upgrade' | 'downgrade' | 'cancel';
    items: CheckoutItem[];
  }>();

  // Show checkout if we need to capture payment, otherwise we just change plans
  const showCheckoutModal = checkoutData?.action === 'upgrade';
  const showChangePlanModal =
    checkoutData?.action === 'downgrade' || checkoutData?.action === 'cancel';
  const preventDowngrade = isCurrentPlanEnterprise;
  const freePlan = plans.find((p) => p?.isFreeTier);

  const onClickChangePlan = ({ item: { planID, name, amount }, action }: ChangePlanArgs) => {
    if (action === 'cancel' && freePlan) {
      setCheckoutData({
        items: [{ planID: freePlan.id, name: freePlan.name, quantity: 1, amount: freePlan.amount }],
        action,
      });
    } else {
      setCheckoutData({ items: [{ planID, name, quantity: 1, amount }], action });
    }
  };

  const onChangePlanSuccess = () => {
    setCheckoutData(undefined);
    router.refresh();
  };

  return (
    <section className="my-14 grid grid-cols-3 gap-2.5">
      {plans.map((plan) => {
        if (!plan || plan.isFreeTier) return;
        const className = cn(
          `rounded-lg border`,
          plan.isActive
            ? 'bg-white border-slate-300 outline outline-2 outline-offset-4 outline-indigo-500'
            : 'bg-slate-100 border-transparent',
          plan.isPremium ? 'bg-slate-900 text-white' : 'text-slate-900'
        );
        return (
          <div key={plan.name} className={className}>
            <BillingPlanOption
              {...plan}
              preventDowngrade={preventDowngrade}
              onClickChangePlan={onClickChangePlan}
            />
            {plan.isUsagePlan && plan.additionalSteps ? (
              <div className="my-1 border-l border-slate-200 px-6">
                <p className="mb-6 text-sm text-slate-500">
                  Additional function steps are billed at the rate of{' '}
                  <strong>${plan.additionalSteps?.cost / 100}</strong> per{' '}
                  <strong>{plan.additionalSteps.quantity}</strong> on the first of every month.
                </p>
              </div>
            ) : (
              ''
            )}
          </div>
        );
      })}
      <div className="col-span-4 mt-8 text-center text-sm font-medium">
        <Button
          href={`${WEBSITE_PRICING_URL}?ref=billing-view-pricing` as Route}
          target="_blank"
          appearance="outlined"
          iconSide="right"
          icon={<ArrowTopRightOnSquareIcon />}
          label="View Pricing Comparison"
        />
      </div>
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
    </section>
  );
}
