import { AlertModal } from '@inngest/components/Modal/AlertModal';

type CurrentEntitlementValues = {
  history?: number;
  metricsExportFreshness?: number;
  metricsExportGranularity?: number;
};

export default function AdvancedObservabilityModal({
  isOpen,
  onClose,
  onSubmit,
  isRemoving,
  addonCost,
  title,
  currentEntitlementValues,
  addonEntitlements,
}: {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: () => void;
  isRemoving: boolean;
  addonCost: number;
  title: string;
  currentEntitlementValues?: CurrentEntitlementValues;
  addonEntitlements?: {
    history: { limit: number };
    metricsExportFreshness: { limit: number };
    metricsExportGranularity: { limit: number };
  };
}) {
  const modalTitle = isRemoving ? `Remove ${title} addon` : `Add ${title.toLowerCase()} to plan`;

  const confirmButtonLabel = isRemoving
    ? 'Remove addon'
    : addonCost > 0
    ? 'Confirm and pay'
    : 'Confirm';

  const removingDescription = `Are you sure you want to remove the ${title.toLowerCase()} addon? You will revert back to your current plan's limits.`;

  const addingDescription = (
    <>
      By clicking Confirm and Pay, the amount of{' '}
      <span className="text-basis font-semibold">${(addonCost / 100).toFixed(2)}</span> will be
      added to your subscription, and your credit card will be charged{' '}
      <span className="text-basis font-semibold">${(addonCost / 100).toFixed(2)} immediately</span>{' '}
      for the remaining days in your billing cycle.
    </>
  );

  return (
    <AlertModal
      isOpen={isOpen}
      onClose={onClose}
      onSubmit={onSubmit}
      title={modalTitle}
      confirmButtonLabel={confirmButtonLabel}
      cancelButtonLabel="Cancel"
      confirmButtonKind={isRemoving ? 'danger' : 'primary'}
      className="w-full max-w-lg"
    >
      {isRemoving ? (
        <div className="space-y-2 p-6">
          <p className="text-muted text-sm leading-relaxed">{removingDescription}</p>
        </div>
      ) : (
        <div className="space-y-2 p-6">
          <p className="text-muted text-sm leading-relaxed">{addingDescription}</p>

          <div className="p-2">
            <div className="border-subtle pb-2">
              <h3 className="text-basis text-md font-semibold">{title}</h3>
            </div>

            <div className="">
              {currentEntitlementValues && addonEntitlements && (
                <>
                  {currentEntitlementValues.history !== undefined && (
                    <div className="flex flex-col justify-between border-y py-2">
                      <span className="text-basis text-sm font-medium">Log retention</span>
                      <div className="flex items-center gap-2">
                        <span className="text-muted text-sm line-through">
                          {`${currentEntitlementValues.history} days`}
                        </span>
                        <span className="text-muted">→</span>
                        <span className="text-basis text-sm font-medium">
                          {`${addonEntitlements.history.limit} days`}
                        </span>
                      </div>
                    </div>
                  )}

                  {currentEntitlementValues.metricsExportGranularity !== undefined && (
                    <div className="flex flex-col justify-between border-b py-3">
                      <span className="text-basis text-sm font-medium">Metrics granularity</span>
                      <div className="flex items-center gap-2">
                        <span className="text-muted text-sm line-through">
                          {`${currentEntitlementValues.metricsExportGranularity / 60} minutes`}
                        </span>
                        <span className="text-muted">→</span>
                        <span className="text-basis text-sm font-medium">
                          {`${addonEntitlements.metricsExportGranularity.limit / 60} minutes`}
                        </span>
                      </div>
                    </div>
                  )}

                  {currentEntitlementValues.metricsExportFreshness !== undefined && (
                    <div className="flex flex-col justify-between border-b py-3">
                      <span className="text-basis text-sm font-medium">Metrics freshness</span>
                      <div className="flex items-center gap-2">
                        <span className="text-muted text-sm line-through">
                          {`${currentEntitlementValues.metricsExportFreshness / 60} minutes`}
                        </span>
                        <span className="text-muted">→</span>
                        <span className="text-basis text-sm font-medium">
                          {`${addonEntitlements.metricsExportFreshness.limit / 60} minutes`}
                        </span>
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>

            <div className="border-subtle flex items-center justify-between border-t pt-4">
              <span className="text-basis">Add on cost</span>
              <span className="text-basis">${(addonCost / 100).toFixed(2)}/mo</span>
            </div>
          </div>
        </div>
      )}
    </AlertModal>
  );
}
