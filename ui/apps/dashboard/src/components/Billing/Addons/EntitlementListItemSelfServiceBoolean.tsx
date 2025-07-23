'use client';

import { Button } from '@inngest/components/Button';

import { pathCreator } from '@/utils/urls';
import { type CurrentEntitlementValues, type Entitlement } from './EntitlementListItem';

function EntitlementDetails({
  currentEntitlementValues,
  addonEntitlements,
}: {
  currentEntitlementValues?: CurrentEntitlementValues;
  addonEntitlements?: {
    history: { limit: number };
    metricsExportFreshness: { limit: number };
    metricsExportGranularity: { limit: number };
  };
}) {
  if (!currentEntitlementValues || !addonEntitlements) return null;

  return (
    <div className="ml-2 mt-2 flex flex-col gap-4 border-l pl-3">
      {currentEntitlementValues.metricsExportGranularity !== undefined && (
        <div className="flex items-end justify-between">
          <div>
            <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
              Metrics Export Granularity
            </p>
            <p className="text-muted mb-1 text-sm italic">Metrics sampling interval</p>
            <div className="text-basis pr-3 text-sm font-medium">
              {`${currentEntitlementValues.metricsExportGranularity / 60} minutes`}
            </div>
          </div>
        </div>
      )}
      {currentEntitlementValues.metricsExportFreshness !== undefined && (
        <div className="flex items-end justify-between">
          <div>
            <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
              Metrics Export Freshness
            </p>
            <p className="text-muted mb-1 text-sm italic">Metrics update frequency</p>
            <div className="text-basis pr-3 text-sm font-medium">
              {`${currentEntitlementValues.metricsExportFreshness / 60} minutes`}
            </div>
          </div>
        </div>
      )}

      {currentEntitlementValues.history !== undefined && (
        <div className="flex items-end justify-between">
          <div>
            <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
              Log History
            </p>
            <p className="text-muted mb-1 text-sm italic">
              Search functions and runs past a certain date.
            </p>
            <div className="text-basis pr-3 text-sm font-medium">
              {`${currentEntitlementValues.history} days`}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function EntitlementListItemSelfServiceBoolean({
  title,
  description,
  tooltip,
  entitlement,
  addonPurchased,
  currentEntitlementValues,
  addonEntitlements,
  onAddClick,
  onRemoveClick,
}: {
  title: string;
  description: string | React.ReactNode;
  tooltip?: React.ReactNode;
  entitlement: Entitlement;
  addonPurchased?: boolean;
  currentEntitlementValues?: CurrentEntitlementValues;
  addonEntitlements?: {
    history: { limit: number };
    metricsExportFreshness: { limit: number };
    metricsExportGranularity: { limit: number };
  };
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
          <Button appearance="outlined" label={`Remove ${title}`} onClick={onRemoveClick} />
        ) : (
          <Button appearance="outlined" label={`Add ${title}`} onClick={onAddClick} />
        )}
      </div>
      <EntitlementDetails
        currentEntitlementValues={currentEntitlementValues}
        addonEntitlements={addonEntitlements}
      />
    </>
  );
}
