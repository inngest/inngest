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
  canIncreaseLimitInCurrentPlan,
  entitlement,
  addon,
  onChange,
  enableSelfServiceFeatureFlag,
}: {
  title: string; // Title of the addon (e.g. Users, Concurrency)
  description?: string;
  tooltipContent?: string | React.ReactNode;
  canIncreaseLimitInCurrentPlan: boolean; // Can this limit be increased on the current billing plan (or does the user need to upgrade to a higher plan)?
  entitlement: {
    currentValue?: number | boolean | null; // Current value of the entitlement relevant to this addon, including plan + any addons + account-level overrides
    displayValue?: string;
    planLimit?: number | null; // The amount of this addon included in the current billing plan
    maxValue?: number; // The maximum amount of (value * quantityPer) that can be purchased
  };
  addon?: {
    addonName: string;
    quantityPer?: number; // The number of units (e.g. concurrency or users) included in one purchase of this addon
    price?: number | null; // Price for one purchase of this addon, in US Cents.
  }; // No addon, or no price, implies self-service is not available.
  onChange?: () => void;
  enableSelfServiceFeatureFlag: boolean;
}) {
  const isPlanEntUnlimited = entitlement.planLimit === undefined || entitlement.planLimit === null; // note: entitlement.planLimit can be 0
  const isAccountEntUnlimited =
    entitlement.currentValue === undefined || entitlement.currentValue === null; // note: entitlement.currentValue can be 0

  // TODO: self service must be unavailable for a given addon if account override is applied for the relevant entitlement
  //       https://linear.app/inngest/issue/INN-4306/self-service-must-be-unavailable-when-account-override-is-applied

  const selfServiceAvailable =
    enableSelfServiceFeatureFlag &&
    !isAccountEntUnlimited &&
    !isPlanEntUnlimited &&
    canIncreaseLimitInCurrentPlan &&
    addon?.price &&
    addon.quantityPer &&
    entitlement.maxValue;

  const priceText =
    !isPlanEntUnlimited &&
    !isAccountEntUnlimited &&
    canIncreaseLimitInCurrentPlan &&
    addon?.price &&
    addon.quantityPer &&
    entitlement.currentValue !== undefined &&
    entitlement.currentValue !== null // note: entitlement.currentValue can be 0
      ? addonPriceStr(title, entitlement.currentValue, addon.quantityPer, addon.price)
      : undefined;

  const planLimitStr =
    isPlanEntUnlimited || isAccountEntUnlimited ? 'unlimited' : entitlement.planLimit!.toString(); // nil-checked at declaration of isPlanEntUnlimited above

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
      {selfServiceAvailable ? (
        <EntitlementListItemSelfService
          title={title}
          description={descriptionText}
          tooltip={tooltip}
          entitlement={{
            currentValue: entitlement.currentValue!, // nil-checked at declaration of selfServiceAvailable above
            displayValue: entitlement.displayValue,
            planLimit: entitlement.planLimit!, // nil-checked at declaration of selfServiceAvailable above
            maxValue: entitlement.maxValue!, // nil-checked at declaration of selfServiceAvailable above
          }}
          addon={{
            price: addon.price!, // nil-checked at declaration of selfServiceAvailable above
            quantityPer: addon.quantityPer!, // nil-checked at declaration of selfServiceAvailable above
            addonName: addon.addonName,
          }}
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
