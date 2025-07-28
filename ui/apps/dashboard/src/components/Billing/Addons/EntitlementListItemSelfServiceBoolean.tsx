'use client';

import { Button } from '@inngest/components/Button';

import { type Entitlement } from './EntitlementListItem';

export default function EntitlementListItemSelfServiceBoolean({
  title,
  description,
  tooltip,
  entitlement,
  addonPurchased,
  buttonText,
  onAddClick,
  onRemoveClick,
}: {
  title: string;
  description: string | React.ReactNode;
  tooltip?: React.ReactNode;
  entitlement: Entitlement;
  addonPurchased?: boolean;
  buttonText?: string;
  onAddClick: () => void;
  onRemoveClick: () => void;
}) {
  return (
    <>
      <div className="flex items-end justify-between">
        <div>
          <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
            {title}
            {tooltip && tooltip}
          </p>
          <p className="text-muted mb-1 text-sm italic">{description}</p>
          {entitlement.displayValue && (
            <div className="text-basis pr-3 text-sm font-medium">{entitlement.displayValue}</div>
          )}
        </div>
        {addonPurchased ? (
          <Button
            appearance="outlined"
            label={buttonText ? `Remove ${buttonText}` : `Remove ${title}`}
            onClick={onRemoveClick}
          />
        ) : (
          <Button
            appearance="outlined"
            label={buttonText ? `Add ${buttonText}` : `Add ${title}`}
            onClick={onAddClick}
          />
        )}
      </div>
    </>
  );
}
