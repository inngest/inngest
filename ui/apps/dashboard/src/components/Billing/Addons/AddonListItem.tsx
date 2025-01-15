'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import CounterInput from '@inngest/components/Forms/CounterInput';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

const planLimit = 5; // TEMP: This is mocked data
const price = 10; // TEMP: This is mocked data
const quantityPer = 1; // TEMP: This is mocked data
const currentValue = 5; // TEMP: This is mocked data
const maxValue = 10000; // TEMP: This is mocked data

export default function AddOn({
  title,
  description,
  value,
  canIncreaseLimitInCurrentPlan,
  tooltipContent,
  selfServiceAvailable,
}: {
  title: string;
  description?: string;
  value?: number | string;
  canIncreaseLimitInCurrentPlan: boolean;
  tooltipContent?: string | React.ReactNode;
  selfServiceAvailable: boolean;
}) {
  const startingValue = Math.max(currentValue, planLimit);
  const [openSelfService, setOpenSelfService] = useState(false);
  const [openConfirmationModal, setOpenConfirmationModal] = useState(false);
  const [inputValue, setInputValue] = useState(startingValue);
  const [inputValid, setInputValid] = useState(true);

  let priceText = `$${price} per ${quantityPer} ${title.toLowerCase()}/${billingPeriod}`;
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
  if (quantityPer === 1) {
    // TODO(cdzombak): extract unit text fn
    let unitDescription = title.toLowerCase();
    if (unitDescription === 'users') {
      unitDescription = 'user';
    }
    priceText = `$${price} per ${unitDescription}/${billingPeriod}`;
  }

  const descriptionText = openSelfService ? (
    <>
      Your plan includes{' '}
      <span className="font-medium lowercase">
        {planLimit} {title}
      </span>
      . Add more for {priceText}.
    </>
  ) : (
    description
  );
  const currentCost =
    inputValue === planLimit ? 0 : Math.ceil((inputValue - planLimit) / quantityPer) * price;

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
          {value && !openSelfService && (
            <p className="text-basis pr-3 text-sm font-medium">{value}</p>
          )}
        </div>

        {!openSelfService && (
          <>
            {selfServiceAvailable && canIncreaseLimitInCurrentPlan ? (
              <Button
                appearance="outlined"
                label={`Add ${title}`}
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
            <CounterInput
              value={inputValue}
              onChange={setInputValue}
              onValid={setInputValid}
              min={planLimit}
              max={maxValue}
              step={quantityPer}
            />
            {inputValid && <p className="text-muted text-sm">Cost: ${currentCost}</p>}
          </div>
          <div className="flex items-center gap-2">
            <Button
              kind="secondary"
              appearance="ghost"
              onClick={() => {
                setOpenSelfService(false);
                setInputValue(Math.max(currentValue, planLimit));
              }}
              label="Cancel"
            />
            <Button
              appearance="outlined"
              disabled={inputValue == startingValue || !inputValid}
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
        <AlertModal
          isOpen={openConfirmationModal}
          onClose={() => setOpenConfirmationModal(false)}
          onSubmit={() => {}}
          title="Add to plan"
          description="Are you sure you want to add this addon to your plan?"
          confirmButtonLabel="Confirm and pay"
          cancelButtonLabel="Cancel"
          confirmButtonKind="primary"
        ></AlertModal>
      )}
    </div>
  );
}
