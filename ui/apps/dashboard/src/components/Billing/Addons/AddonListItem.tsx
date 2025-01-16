'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import CounterInput from '@inngest/components/Forms/CounterInput';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

const billingPeriod = 'month'; // TODO(cdzombak): need this from the backend

function unitDescriptionFromTitle(title: string) {
  let result = title.toLowerCase();
  if (result === 'users') {
    result = 'user';
  }
  return result;
}

export default function AddOn({
  title,
  description,
  tooltipContent,
  value,
  displayValue,
  planLimit,
  maxValue,
  canIncreaseLimitInCurrentPlan,
  selfServiceAvailable,
  quantityPer,
  price,
}: {
  title: string; // Name/Title of the addon (e.g. Users, Concurrency)
  description?: string;
  tooltipContent?: string | React.ReactNode;
  value?: number | boolean; // Current value of the entitlement relevant to this addon, including plan + any addons + account-level overrides
  displayValue?: string; // Display value of the entitlement relevant to this addon (including e.g. formatting)
  planLimit: number | boolean; // The amount of this addon included in the current billing plan
  maxValue: number; // The maximum amount of (value * quantityPer) that can be purchased
  canIncreaseLimitInCurrentPlan: boolean; // Can this limit be increased on the current billing plan (or does the user need to upgrade to a higher plan)?
  selfServiceAvailable: boolean; // Is this addon available (for this account) via self-service or will they need to contact us?
  quantityPer: number; // The number of units (e.g. concurrency or users) included in one purchase of this addon
  price?: number; // Price for one purchase of this addon, in US Cents
}) {
  const [openSelfService, setOpenSelfService] = useState(false);
  const [openConfirmationModal, setOpenConfirmationModal] = useState(false);
  const [inputValid, setInputValid] = useState(true);

  const useSwitchInput = typeof value === 'boolean';
  const useNumericInput = typeof value === 'number' && typeof planLimit === 'number';
  if (!useSwitchInput && !useNumericInput) {
    throw new Error('AddOn requires either a boolean or numeric value');
  }

  const startingNumericInputValue = useNumericInput ? Math.max(value, planLimit) : 0;
  const [numericInputValue, setNumericInputValue] = useState(startingNumericInputValue);

  const priceDollars = price ? (price / 100).toFixed(2) : undefined;
  let priceText = !price
    ? undefined
    : `$${priceDollars} per ${quantityPer} ${title.toLowerCase()}/${billingPeriod}`;
  if (quantityPer === 1) {
    priceText = `$${priceDollars} per ${unitDescriptionFromTitle(title)}/${billingPeriod}`;
  } else if (useSwitchInput) {
    priceText = `$${priceDollars} per ${billingPeriod}`;
  }

  const descriptionText = openSelfService ? (
    useSwitchInput && !planLimit ? (
      <>
        Your plan does not include{' '}
        <span className="font-medium lowercase">
          {planLimit} {title}
        </span>
        by default.{price && ` Add it for ${priceText}.`}
      </>
    ) : (
      <>
        Your plan includes{' '}
        <span className="font-medium lowercase">
          {planLimit} {title}
        </span>
        .{price && ` Add more for ${priceText}.`}
      </>
    )
  ) : (
    description
  );

  const cost = !price
    ? undefined
    : numericInputValue === planLimit
    ? 0
    : (
        (Math.ceil(
          (numericInputValue - (typeof planLimit == 'boolean' ? 1 : planLimit)) / quantityPer
        ) *
          price) /
        100
      ).toFixed(2);

  const confirmationTitle =
    value === planLimit
      ? `Add ${title.toLowerCase()} to plan`
      : `Change ${title.toLowerCase()} addon`;
  const addedDescription = useNumericInput
    ? `Your new charge for ${numericInputValue} ${title.toLowerCase()} will be $${cost} per ${billingPeriod}.`
    : '';

  return (
    <div className="mb-5">
      <div className="flex items-end justify-between">
        <div>
          <p className="text-subtle mb-1 flex items-center gap-1 text-sm font-medium">
            {title}
            {tooltipContent && (
              <Tooltip>
                <TooltipTrigger>
                  <RiInformationLine className="text-light h-4 w-4" />
                </TooltipTrigger>
                <TooltipContent className="whitespace-pre-line text-left">
                  {tooltipContent}
                </TooltipContent>
              </Tooltip>
            )}
          </p>
          {description && <p className="text-muted mb-0.5 text-sm italic">{descriptionText}</p>}
          {displayValue && !openSelfService && (
            <p className="text-basis pr-3 text-sm font-medium">{displayValue}</p>
          )}
        </div>

        {!openSelfService && (
          <>
            {selfServiceAvailable && canIncreaseLimitInCurrentPlan ? (
              <Button
                appearance="outlined"
                label={value === planLimit ? `Add ${title}` : `Update ${title}`}
                onClick={() => {
                  setOpenSelfService(true);
                  setInputValid(true);
                }}
              />
            ) : (
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
            )}
          </>
        )}
      </div>
      {openSelfService && (
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-4">
            {useNumericInput && (
              <CounterInput
                value={numericInputValue}
                onChange={setNumericInputValue}
                onValid={setInputValid}
                min={planLimit}
                max={maxValue}
                step={quantityPer}
              />
            )}
            {inputValid && <p className="text-muted text-sm">Cost: ${cost}</p>}
          </div>
          <div className="flex items-center gap-2">
            <Button
              kind="secondary"
              appearance="ghost"
              onClick={() => {
                setOpenSelfService(false);
                setNumericInputValue(startingNumericInputValue);
              }}
              label="Cancel"
            />
            <Button
              appearance="outlined"
              disabled={numericInputValue == startingNumericInputValue || !inputValid}
              onClick={() => {
                setOpenConfirmationModal(true);
                setOpenSelfService(false);
              }}
              label="Update"
            />
          </div>
        </div>
      )}
      {openConfirmationModal && (
        // TODO(cdzombak): include addon name, price, quantity/resolved amount in confirmation modal
        // TODO(cdzombak): actually commit the change
        <AlertModal
          isOpen={openConfirmationModal}
          onClose={() => setOpenConfirmationModal(false)}
          onSubmit={() => {}}
          title={confirmationTitle}
          description={
            'Are you sure you want to apply this change to your plan? ' + addedDescription
          }
          confirmButtonLabel="Confirm and pay"
          cancelButtonLabel="Cancel"
          confirmButtonKind="primary"
        ></AlertModal>
      )}
    </div>
  );
}
