import { Pill } from '@inngest/components/Pill/Pill';
import { RiCheckLine } from '@remixicon/react';

import { isActive, isTrialPlan, processPlan, type Plan } from '@/components/Billing/Plans/utils';
import UpgradeButton from './UpgradeButton';

export function VerticalPlanCard({
  plan,
  currentPlan,
  onPlanChange,
}: {
  plan: Plan;
  currentPlan: Plan;
  onPlanChange: () => void;
}) {
  const transformedPlan = processPlan(plan);
  const displayTrialPill = isActive(currentPlan, plan) && isTrialPlan(currentPlan);

  return (
    <div className="border-muted bg-canvasBase rounded-md border p-6">
      <h4 className="text-basis mb-2 flex items-center gap-2 text-2xl font-medium">
        {transformedPlan.name}
        {displayTrialPill && <Pill>Trial</Pill>}
      </h4>
      <div className="mb-1 text-xs font-bold uppercase">From</div>
      <div className="text-2xl">
        <span className="text-4xl font-medium">{transformedPlan.price}</span>/
        {transformedPlan.billingPeriod}
      </div>

      <UpgradeButton plan={plan} currentPlan={currentPlan} onPlanChange={onPlanChange} />
      <hr className="mb-6" />
      <ul className="flex flex-col">
        {transformedPlan.features.map((feature, i) => (
          <li key={i} className={`flex items-start gap-2 py-2 first:pt-0 last:pb-0`}>
            <div className="flex h-6 items-center">
              <RiCheckLine className="text-primary-subtle h-4 w-4" />
            </div>
            {feature}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function HorizontalPlanCard({
  plan,
  currentPlan,
  onPlanChange,
}: {
  plan: Plan;
  currentPlan: Plan;
  onPlanChange: () => void;
}) {
  const transformedPlan = processPlan(plan);
  const displayTrialPill = isActive(currentPlan, plan) && isTrialPlan(currentPlan);
  // Split features into two columns
  const halfwayIndex = Math.ceil(transformedPlan.features.length / 2);
  const firstColumn = transformedPlan.features.slice(0, halfwayIndex);
  const secondColumn = transformedPlan.features.slice(halfwayIndex);

  return (
    <div className="border-muted bg-canvasBase grid grid-cols-3 items-center gap-12 rounded-md border p-6">
      <div>
        <h4 className="text-basis mb-2 flex items-center gap-2 text-2xl font-medium">
          {transformedPlan.name}
          {displayTrialPill && <Pill>Trial</Pill>}
        </h4>
        <UpgradeButton plan={plan} currentPlan={currentPlan} onPlanChange={onPlanChange} />
      </div>

      <div className="col-span-2 grid grid-cols-2 gap-8">
        {/* First Column */}
        <ul className="flex flex-col">
          {firstColumn.map((feature, i) => (
            <li key={i} className="flex items-start gap-2 py-2 first:pt-0 last:pb-0">
              <div className="flex h-6 items-center">
                <RiCheckLine className="text-primary-subtle h-4 w-4" />
              </div>
              {feature}
            </li>
          ))}
        </ul>

        {/* Second Column */}
        <ul className="flex flex-col">
          {secondColumn.map((feature, i) => (
            <li key={i + halfwayIndex} className="flex items-start gap-2 py-2 first:pt-0 last:pb-0">
              <div className="flex h-6 items-center">
                <RiCheckLine className="text-primary-subtle h-4 w-4" />
              </div>
              {feature}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
