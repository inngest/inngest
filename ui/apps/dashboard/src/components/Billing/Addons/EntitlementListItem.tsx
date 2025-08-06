'use client';

import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { cn } from '@inngest/components/utils/classNames';
import { RiInformationLine } from '@remixicon/react';

import EntitlementListItemSelfService from '@/components/Billing/Addons/EntitlementListItemSelfService';
import { PlanNames } from '@/components/Billing/Plans/utils';
import { pathCreator } from '@/utils/urls';

export default function EntitlementListItem({
  increaseInHigherPlan = true,
  planName,
  title,
  description,
  tooltipContent,
  entitlement,
  addon,
  onChange,
}: {
  increaseInHigherPlan?: boolean;
  planName: string;
  title: string; // Title of the addon (e.g. Users, Concurrency)
  description: string;
  tooltipContent?: string | React.ReactNode;
  entitlement: {
    currentValue: number | boolean | null; // Current value of the entitlement relevant to this addon, including plan + any addons + account-level overrides
    displayValue: string | React.ReactNode;
  };
  addon?: {
    available: boolean;
    name: string;
    baseValue: number | null;
    maxValue: number;
    quantityPer: number; // The number of units (e.g. concurrency or users) included in one purchase of this addon
    price: number | null; // Price for one purchase of this addon, in US Cents.
    purchased?: boolean;
  }; // No addon, or no price, implies self-service is not available.
  onChange?: () => void;
}) {
  const [highlight] = useSearchParam('highlight');

  const highlighted = highlight === title.toLowerCase().replace(/ /g, '-');
  const tooltip = tooltipContent ? (
    <Tooltip>
      <TooltipTrigger>
        <RiInformationLine className="text-light h-4 w-4" />
      </TooltipTrigger>
      <TooltipContent className="whitespace-pre-line text-left">{tooltipContent}</TooltipContent>
    </Tooltip>
  ) : null;

  let content;
  if (
    addon &&
    addon.available &&
    addon.baseValue !== null &&
    addon.price !== null &&
    entitlement.currentValue !== null
  ) {
    // The user can increase this entitlement by purchasing an addon.

    content = (
      <EntitlementListItemSelfService
        title={title}
        description={description}
        tooltip={tooltip}
        entitlement={{
          currentValue: entitlement.currentValue,
          displayValue: entitlement.displayValue,
          planLimit: addon.baseValue,
          maxValue: addon.maxValue,
        }}
        addon={{
          price: addon.price,
          quantityPer: addon.quantityPer,
          addonName: addon.name,
        }}
        addonPurchased={addon.purchased}
        onChange={onChange}
      />
    );
  } else {
    // The user cannot increase this entitlement by purchasing an addon.

    const isCustomPlan = planName.toLowerCase().includes('enterprise');
    const isHighestPlan = planName === PlanNames.Pro;

    let contactHumanToIncrease = false;
    if (isHighestPlan) {
      // No higher plan to upgrade to.
      contactHumanToIncrease = true;
    } else if (isCustomPlan) {
      // Enterprise accounts get entitlement increases by talking to a human.
      contactHumanToIncrease = true;
    } else if (!increaseInHigherPlan) {
      // Upgrading to a higher plan won't increase this entitlement.
      contactHumanToIncrease = true;
    } else if (addon && addon.price === null) {
      // If there isn't an addon price then they need to talk to a human. This
      // is probably rarely hit (e.g. when we haven't added a Stripe price yet).
      contactHumanToIncrease = true;
    }

    content = (
      <div className="flex items-end justify-between">
        <div>
          <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
            {title}
            {tooltip && tooltip}
          </p>
          <p className="text-muted mb-1 text-sm italic">{description}</p>
          <div className="text-basis pr-3 text-sm font-medium">{entitlement.displayValue}</div>
        </div>
        <Button
          appearance="ghost"
          label={contactHumanToIncrease ? 'Contact us' : 'Upgrade plan'}
          href={
            contactHumanToIncrease
              ? pathCreator.support({
                  ref: `app-billing-page-overview-addon-${title.toLowerCase().replace(/ /g, '-')}`,
                })
              : pathCreator.billing({
                  tab: 'plans',
                  ref: `app-billing-page-overview-addon-${title.toLowerCase().replace(/ /g, '-')}`,
                })
          }
        />
      </div>
    );
  }

  return (
    <div
      className={cn(
        'mb-5 transition-colors duration-300',
        highlighted && 'bg-primary-subtle/10 border-primary-subtle/20 rounded-md border p-3'
      )}
    >
      {content}
    </div>
  );
}
