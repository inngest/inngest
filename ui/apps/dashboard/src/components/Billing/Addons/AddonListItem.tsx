'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import CounterInput from '@inngest/components/Forms/CounterInput';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

export default function AddOn({
  title,
  description,
  value,
  canIncreaseLimitInCurrentPlan,
  tooltipContent,
  // TEMP: This is a temporary prop to show the self-service option
  selfServiceAvailable = true,
}: {
  title: string;
  description?: string;
  value?: number | string;
  canIncreaseLimitInCurrentPlan: boolean;
  tooltipContent?: string | React.ReactNode;
  selfServiceAvailable?: boolean;
}) {
  const [openSelfService, setOpenSelfService] = useState(false);

  const planLimit = 5; // TEMP: This is mocked data
  const price = 10; // TEMP: This is mocked data
  const quantityPer = 1; // TEMP: This is mocked data

  const priceText = `$${price} per ${quantityPer}`;
  const descriptionText = openSelfService ? (
    <>
      Your plan comes with{' '}
      <span className="font-medium lowercase">
        {planLimit} {title} by default
      </span>
      . Additional cost is {priceText}
    </>
  ) : (
    description
  );

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
                onClick={() => setOpenSelfService(true)}
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
          <div className="flex items-center gap-4">
            <CounterInput />
            <p className="text-muted text-sm">Cost: $50</p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              kind="secondary"
              appearance="ghost"
              onClick={() => {
                setOpenSelfService(false);
              }}
              label="Cancel"
            />
            <Button
              appearance="outlined"
              onClick={() => {
                setOpenSelfService(false);
              }}
              label="Add"
            />
          </div>
        </div>
      )}
    </div>
  );
}
