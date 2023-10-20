'use client';

import { type Route } from 'next';
import { Button } from '@inngest/components/Button';

import cn from '@/utils/cn';
import { WEBSITE_CONTACT_URL } from '@/utils/urls';
import { type CheckoutItem } from './CheckoutModal';
import PlanBadge from './PlanBadge';

export type ChangePlanArgs = {
  item: CheckoutItem;
  action: 'upgrade' | 'downgrade' | 'cancel';
};

type BillingPlanOptionProps = {
  id: string;
  name: string;
  amount: number;
  isLowerTierPlan: boolean;
  isActive: boolean;
  isPremium: boolean;
  usagePercentage: number;
  features: Record<string, unknown>;
  preventDowngrade: boolean;
  onClickChangePlan: (args: ChangePlanArgs) => void;
};

type FeatureRow = { key: string; label: string; showFreeTier?: boolean; unit?: string };
export const featureRows: FeatureRow[] = [
  { key: 'actions', label: 'Function Steps', showFreeTier: true },
  { key: 'concurrency', label: 'Concurrent Functions', showFreeTier: true },
  { key: 'log_retention', label: 'History', showFreeTier: true, unit: 'days' },
  { key: 'events', label: 'Events' },
  { key: 'users', label: 'Seats' },
  { key: 'support', label: 'Support' },
];

function isPrimitive(value: unknown): value is boolean | number | string {
  return typeof value === 'boolean' || typeof value === 'number' || typeof value === 'string';
}

export default function BillingPlanOption({
  id,
  name,
  amount,
  isLowerTierPlan,
  isActive,
  isPremium,
  usagePercentage,
  features,
  preventDowngrade,
  onClickChangePlan,
}: BillingPlanOptionProps) {
  const cost = amount === Infinity ? 'Custom Pricing' : `$${amount / 100}`;
  const badgeText = isActive ? 'Current Plan' : isLowerTierPlan ? 'Downgrade' : 'Upgrade';

  const onClick = () => {
    onClickChangePlan({
      action: isActive ? 'cancel' : isLowerTierPlan ? 'downgrade' : 'upgrade',
      item: { planID: id, name, quantity: 1, amount },
    });
  };

  return (
    <div className="p-6">
      <Row className="mb-4">
        <PlanBadge variant={isActive || !isLowerTierPlan ? 'primary' : 'default'}>
          {badgeText}
        </PlanBadge>
        <div className="text-lg font-bold">{cost}</div>
      </Row>

      <Row className="mb-1.5">
        <h2 className="text-[1.375rem] font-semibold">{name}</h2>
        <div className="text-sm font-medium">
          {/* {usagePercentage >= 100 ? 'Exceeding plan limits' : 'Usage'} */}
        </div>
      </Row>
      {/* Disable usage bar until API is ready */}
      {false && !isPremium ? (
        <FunctionUsageBar percentage={usagePercentage} />
      ) : (
        <span className="block h-1.5">{/* Spacer */}</span>
      )}

      <FeatureRows featureRows={featureRows} features={features} isPremium={isPremium} />

      <Button
        href={isPremium ? (`${WEBSITE_CONTACT_URL}?ref=billing-enterprise` as Route) : undefined}
        target={isPremium ? '_blank' : undefined}
        btnAction={!isPremium ? onClick : undefined}
        className="mt-5 w-full"
        appearance={isActive || isLowerTierPlan ? 'outlined' : 'solid'}
        kind={isActive ? 'danger' : isLowerTierPlan ? 'default' : 'primary'}
        disabled={preventDowngrade && isLowerTierPlan}
        label={
          isLowerTierPlan
            ? 'Downgrade Plan'
            : isActive && isPremium
            ? 'Contact Account Manager'
            : isActive
            ? 'Cancel Subscription'
            : isPremium
            ? 'Get In Touch'
            : 'Upgrade'
        }
      />
    </div>
  );
}

export function FeatureRows({
  featureRows = [],
  features,
  isPremium,
}: {
  featureRows: FeatureRow[];
  features: BillingPlanOptionProps['features'];
  isPremium: boolean;
}) {
  return (
    <>
      {featureRows.map(({ label, key, unit }, idx) => {
        const featureValue = features[key];
        if (!isPrimitive(featureValue)) {
          return null;
        }

        return (
          <Row key={key} className={cn('my-3', idx === 0 && 'mt-6')}>
            <span className="text-sm font-medium text-slate-400">{label}</span>
            <span className={cn(`font-semibold text-slate-800`, isPremium && 'text-slate-50')}>
              {featureValue}
              {unit ? ` ${unit}` : ''}
            </span>
          </Row>
        );
      })}
    </>
  );
}

export function Row({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn('flex items-center justify-between', className)}>{children}</div>;
}

function FunctionUsageBar({ percentage }: { percentage: number }) {
  const className = cn(
    'block bg-indigo-500 rounded-full',
    percentage >= 99 ? 'bg-red-500' : percentage >= 80 ? 'bg-amber-400' : null
  );
  return (
    <div className="flex h-1.5 rounded-full bg-slate-300">
      <span style={{ flexBasis: `${percentage}%` }} className={className}></span>
    </div>
  );
}
