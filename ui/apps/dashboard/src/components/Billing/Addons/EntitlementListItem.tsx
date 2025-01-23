'use client';

import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine } from '@remixicon/react';

import EntitlementListItemSelfService from '@/components/Billing/Addons/EntitlementListItemSelfService';
import { addonPriceStr } from '@/components/Billing/Addons/pricing_help';
import { pathCreator } from '@/utils/urls';

export default function EntitlementListItem({
  title,
  description,
  tooltipContent,
  selfServiceAvailable,
  canIncreaseLimitInCurrentPlan,
  entitlement,
  addon,
  onChange,
}: {
  title: string; // Title of the addon (e.g. Users, Concurrency)
  description?: string;
  tooltipContent?: string | React.ReactNode;
  selfServiceAvailable: boolean; // Is an addon available (for this account) via self-service or will they need to contact us?
  canIncreaseLimitInCurrentPlan: boolean; // Can this limit be increased on the current billing plan (or does the user need to upgrade to a higher plan)?
  entitlement: {
    currentValue?: number | boolean; // Current value of the entitlement relevant to this addon, including plan + any addons + account-level overrides
    displayValue?: string;
    planLimit?: number; // The amount of this addon included in the current billing plan
    maxValue?: number; // The maximum amount of (value * quantityPer) that can be purchased
  };
  addon?: {
    addonName: string;
    quantityPer: number; // The number of units (e.g. concurrency or users) included in one purchase of this addon
    price: number; // Price for one purchase of this addon, in US Cents
  };
  onChange?: () => void;
}) {
  const isPlanEntUnlimited = entitlement.planLimit === undefined || entitlement.planLimit === -1;

  if (entitlement.currentValue === undefined) {
    entitlement.currentValue = -1;
  }
  if (!canIncreaseLimitInCurrentPlan || isPlanEntUnlimited) {
    selfServiceAvailable = false;
  }
  if (selfServiceAvailable && !addon) {
    console.error('EntitlementListItem: addon is required when selfServiceAvailable is true');
    selfServiceAvailable = false;
  }
  if (selfServiceAvailable && !entitlement.maxValue) {
    console.error(
      'EntitlementListItem: entitlement.maxValue is required when selfServiceAvailable is true'
    );
    selfServiceAvailable = false;
  }

  const priceText =
    !isPlanEntUnlimited && canIncreaseLimitInCurrentPlan && addon
      ? addonPriceStr(title, entitlement, addon)
      : undefined;
  const planLimitStr = isPlanEntUnlimited ? 'unlimited' : entitlement.planLimit!.toString();
  const planIncludesStr =
    typeof entitlement.currentValue === 'boolean' ? title : planLimitStr + ' ' + title;
  const descriptionText = description ? (
    description
  ) : (
    <>
      Your plan{' '}
      {typeof entitlement.currentValue === 'boolean' && !entitlement.currentValue
        ? 'does not include'
        : 'includes'}{' '}
      <span className="font-medium lowercase">{planIncludesStr}</span>.
      {priceText &&
        ` Add ${typeof entitlement.currentValue === 'boolean' ? 'it' : 'more'} for ${priceText}.`}
    </>
  );

  const tooltip = tooltipContent ? (
    <Tooltip>
      <TooltipTrigger>
        <RiInformationLine className="text-light h-4 w-4" />
      </TooltipTrigger>
      <TooltipContent className="whitespace-pre-line text-left">{tooltipContent}</TooltipContent>
    </Tooltip>
  ) : null;

  return (
    <div className="mb-5">
      {selfServiceAvailable && addon && entitlement.maxValue && entitlement.planLimit ? (
        <EntitlementListItemSelfService
          title={title}
          description={descriptionText}
          tooltip={tooltip}
          entitlement={{
            currentValue: entitlement.currentValue,
            planLimit: entitlement.planLimit,
            maxValue: entitlement.maxValue,
          }}
          addon={addon}
          onChange={onChange}
        />
      ) : (
        <div className="flex items-end justify-between">
          <div>
            <p className="text-subtle mb-1 flex items-center gap-1 text-sm font-medium">
              {title}
              {tooltip && tooltip}
            </p>
            <p className="text-muted mb-0.5 text-sm italic">{descriptionText}</p>
            {entitlement.displayValue && (
              <p className="text-basis pr-3 text-sm font-medium">{entitlement.displayValue}</p>
            )}
          </div>
          <Button
            appearance="ghost"
            label={canIncreaseLimitInCurrentPlan ? 'Contact us' : 'Upgrade'}
            href={
              canIncreaseLimitInCurrentPlan
                ? pathCreator.support({
                    ref: `app-billing-page-overview-addon-${title
                      .toLowerCase()
                      .replace(/ /g, '-')}`,
                  })
                : pathCreator.billing({
                    tab: 'plans',
                    ref: `app-billing-page-overview-addon-${title
                      .toLowerCase()
                      .replace(/ /g, '-')}`,
                  })
            }
          />
        </div>
      )}
    </div>
  );
}
