'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { RiAlertFill } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import EntitlementListItemSelfServiceBoolean from '@/components/Billing/Addons/EntitlementListItemSelfServiceBoolean';
import EntitlementListItemSelfServiceNumeric from '@/components/Billing/Addons/EntitlementListItemSelfServiceNumeric';
import { addonQtyCostString } from '@/components/Billing/Addons/pricing_help';
import { graphql } from '@/gql';
import { type CurrentEntitlementValues, type Entitlement } from './EntitlementListItem';

const UpdateAccountAddonQuantityDocument = graphql(`
  mutation UpdateAccountAddonQuantity($addonName: String!, $quantity: Int!) {
    updateAccountAddonQuantity(addonName: $addonName, quantity: $quantity) {
      purchaseCount
    }
  }
`);

export default function EntitlementListItemSelfService({
  title,
  description,
  tooltip,
  entitlement,
  addon,
  addonPurchased,
  currentEntitlementValues,
  onChange,
}: {
  title: string;
  description: string | React.ReactNode;
  tooltip?: React.ReactNode;
  entitlement: Entitlement;
  addon: {
    addonName: string;
    quantityPer: number;
    price: number;
    entitlements?: {
      history: {
        limit: number;
      };
      metricsExportFreshness: {
        limit: number;
      };
      metricsExportGranularity: {
        limit: number;
      };
    };
  };
  addonPurchased?: boolean;
  currentEntitlementValues?: CurrentEntitlementValues;
  onChange?: () => void;
}) {
  const router = useRouter();

  const [openSelfService, setOpenSelfService] = useState(false);
  const [openConfirmationModal, setOpenConfirmationModal] = useState(false);
  const [addonCost, setAddonCost] = useState(0);
  const [addonQty, setAddonQty] = useState(0);
  const [, updateAccountAddonQuantity] = useMutation(UpdateAccountAddonQuantityDocument);
  const [err, setErr] = useState<String | null>(null);

  const switchInput = typeof entitlement.currentValue === 'boolean';
  const numericInput = typeof entitlement.currentValue === 'number';

  const addonCostStr = addonQtyCostString(addonQty, addon);

  const addonConfirmTitle =
    entitlement.currentValue === entitlement.planLimit || (switchInput && !entitlement.currentValue)
      ? `Add ${title.toLowerCase()} to plan`
      : `Change ${title.toLowerCase()} addon`;

  const addonConfirmDescription = numericInput
    ? `Your new charge for ${
        addonQty * addon.quantityPer
      } ${title.toLowerCase()} will be ${addonCostStr}.`
    : `Your new charge for ${title.toLowerCase()} will be ${addonCostStr}.`;

  const handleSubmit = async () => {
    setOpenConfirmationModal(false);
    const updateResult = await updateAccountAddonQuantity({
      addonName: addon.addonName,
      quantity: addonQty,
    });
    if (updateResult.error) {
      console.error(updateResult.error.message);
      setErr(updateResult.error.message);
    } else {
      setErr(null);
      if (onChange) {
        onChange();
      }
      router.refresh();
      toast.success(`Addon updated successfully`);
    }
  };

  if (switchInput) {
    const handleBooleanAddClick = () => {
      const quantity = 1; // Boolean addons are always quantity 1
      const cost = quantity * addon.price;
      setAddonQty(quantity);
      setAddonCost(cost);
      setOpenConfirmationModal(true);
    };

    return (
      <>
        {err && (
          <p className="text-error mb-2 text-xs">
            <RiAlertFill className="-mt-0.5 inline h-4" /> Failed to update addon.{' '}
            <a href="/support" className="underline">
              Contact support
            </a>{' '}
            if this problem persists.
          </p>
        )}
        <EntitlementListItemSelfServiceBoolean
          title={title}
          description={description}
          tooltip={tooltip}
          entitlement={entitlement}
          addonPurchased={addonPurchased}
          currentEntitlementValues={currentEntitlementValues}
          addonEntitlements={addon.entitlements}
          onAddClick={handleBooleanAddClick}
        />
        {openConfirmationModal && (
          <AlertModal
            isOpen={openConfirmationModal}
            onClose={() => setOpenConfirmationModal(false)}
            onSubmit={handleSubmit}
            title="Add to plan"
            confirmButtonLabel={addonCost > 0 ? 'Confirm and pay' : 'Confirm'}
            cancelButtonLabel="Cancel"
            confirmButtonKind="primary"
            className="w-full max-w-lg"
          >
            <div className="space-y-2 p-6">
              <p className="text-muted text-sm leading-relaxed">
                By clicking Confirm and Pay, the amount of{' '}
                <span className="text-basis font-semibold">${(addonCost / 100).toFixed(2)}</span>{' '}
                will be added to your subscription, and your credit card will be charged{' '}
                <span className="text-basis font-semibold">
                  ${(addonCost / 100).toFixed(2)} immediately
                </span>{' '}
                for the remaining days in your billing cycle.
              </p>
              <div className="p-2">
                <div className="border-subtle pb-2">
                  <h3 className="text-basis text-md font-semibold">{title}</h3>
                </div>

                <div className="">
                  {typeof entitlement.currentValue === 'number' && entitlement.planLimit && (
                    <div className="flex flex-col justify-between py-3">
                      <span className="text-basis text-sm font-medium">{title}</span>
                      <div className="flex items-center gap-2">
                        <span className="text-muted text-sm line-through">
                          {entitlement.planLimit.toLocaleString()}
                        </span>
                        <span className="text-muted">→</span>
                        <span className="text-basis text-sm font-medium">
                          {(entitlement.planLimit + addonQty * addon.quantityPer).toLocaleString()}
                        </span>
                      </div>
                    </div>
                  )}

                  {currentEntitlementValues && addon.entitlements && (
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
                              {`${addon.entitlements.history.limit} days`}
                            </span>
                          </div>
                        </div>
                      )}

                      {currentEntitlementValues.metricsExportGranularity !== undefined && (
                        <div className="flex flex-col justify-between border-b py-3">
                          <span className="text-basis text-sm font-medium">
                            Metrics granularity
                          </span>
                          <div className="flex items-center gap-2">
                            <span className="text-muted text-sm line-through">
                              {`${currentEntitlementValues.metricsExportGranularity / 60} minutes`}
                            </span>
                            <span className="text-muted">→</span>
                            <span className="text-basis text-sm font-medium">
                              {`${addon.entitlements.metricsExportGranularity.limit / 60} minutes`}
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
                              {`${addon.entitlements.metricsExportFreshness.limit / 60} minutes`}
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
          </AlertModal>
        )}
      </>
    );
  }

  return (
    <>
      <div className="flex items-end justify-between">
        <div>
          <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
            {title}
            {tooltip && tooltip}
          </p>
          <p className="text-muted mb-1 text-sm italic">{description}</p>
          {err && (
            <p className="text-error text-xs">
              <RiAlertFill className="-mt-0.5 inline h-4" /> Failed to update addon.{' '}
              <a href="/support" className="underline">
                Contact support
              </a>{' '}
              if this problem persists.
            </p>
          )}
          {entitlement.displayValue && !openSelfService ? (
            <div className="text-basis pr-3 text-sm font-medium">{entitlement.displayValue}</div>
          ) : null}
        </div>
        {!openSelfService && (
          <Button
            appearance="outlined"
            label={
              entitlement.currentValue === entitlement.planLimit
                ? `Add ${title}`
                : `Update ${title}`
            }
            onClick={() => {
              setOpenSelfService(true);
              setErr(null);
            }}
          />
        )}
      </div>
      {openSelfService &&
        numericInput &&
        typeof entitlement.currentValue === 'number' &&
        entitlement.planLimit != null &&
        entitlement.maxValue != null && (
          <EntitlementListItemSelfServiceNumeric
            entitlement={{
              currentValue: entitlement.currentValue,
              planLimit: entitlement.planLimit,
              maxValue: entitlement.maxValue,
            }}
            addon={addon}
            onCancel={() => {
              setOpenSelfService(false);
              setErr(null);
            }}
            onSubmit={(qty: number, cost: number) => {
              setAddonQty(qty);
              setAddonCost(cost);
              setOpenConfirmationModal(true);
              setOpenSelfService(false);
            }}
          />
        )}
      {openConfirmationModal && (
        <AlertModal
          isOpen={openConfirmationModal}
          onClose={() => setOpenConfirmationModal(false)}
          onSubmit={handleSubmit}
          title={addonConfirmTitle}
          description={
            'Are you sure you want to apply this change to your plan? ' + addonConfirmDescription
          }
          confirmButtonLabel={(addonCost || 0) > 0 ? 'Confirm and pay' : 'Confirm'}
          cancelButtonLabel="Cancel"
          confirmButtonKind="primary"
        />
      )}
    </>
  );
}
