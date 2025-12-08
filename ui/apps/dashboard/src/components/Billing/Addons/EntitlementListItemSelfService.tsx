import { useState } from 'react';

import { Button } from '@inngest/components/Button/NewButton';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { RiAlertFill } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import AdvancedObservabilityComponent from '@/components/Billing/Addons/AdvancedObservabilityModal';
import EntitlementListItemSelfServiceNumeric from '@/components/Billing/Addons/EntitlementListItemSelfServiceNumeric';
import { addonQtyCostString } from '@/components/Billing/Addons/pricing_help';
import { graphql } from '@/gql';
import SlackChannelComponent from './SlackChannelModal';
import { useNavigate } from '@tanstack/react-router';

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
  onChange,
}: {
  title: string;
  description: string | React.ReactNode;
  tooltip?: React.ReactNode;
  entitlement: {
    currentValue: number | boolean;
    displayValue?: string | React.ReactNode;
    planLimit: number;
    maxValue: number;
  };
  addon: {
    addonName: string;
    quantityPer: number;
    price: number;
  };
  addonPurchased?: boolean;
  onChange?: () => void;
}) {
  const navigate = useNavigate();

  const [openSelfService, setOpenSelfService] = useState(false);
  const [openConfirmationModal, setOpenConfirmationModal] = useState(false);
  const [addonCost, setAddonCost] = useState(0);
  const [addonQty, setAddonQty] = useState(0);
  const [isRemoving, setIsRemoving] = useState(false);
  const [, updateAccountAddonQuantity] = useMutation(
    UpdateAccountAddonQuantityDocument,
  );
  const [err, setErr] = useState<string | null>(null);

  const switchInput = typeof entitlement.currentValue === 'boolean';
  const numericInput = typeof entitlement.currentValue === 'number';

  const isAdvancedObservability =
    title === 'Log retention' ||
    title === 'Metrics granularity' ||
    title === 'Metrics freshness';

  const isDedicatedSlackChannel = title === 'Dedicated Slack Channel';

  const addonCostStr = addonQtyCostString(addonQty, addon);

  const addonConfirmTitle =
    entitlement.currentValue === entitlement.planLimit ||
    (switchInput && !entitlement.currentValue)
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
      navigate({ to: '.', replace: true });
      toast.success(`Addon ${isRemoving ? 'removed' : 'updated'} successfully`);
    }
    setIsRemoving(false);
  };

  if (switchInput) {
    if (isAdvancedObservability) {
      return (
        <AdvancedObservabilityComponent
          title={title}
          description={description}
          tooltip={tooltip}
          entitlement={entitlement}
          addon={addon}
          addonPurchased={addonPurchased}
          onChange={onChange}
        />
      );
    } else if (isDedicatedSlackChannel) {
      return (
        <SlackChannelComponent
          title={title}
          description={description}
          tooltip={tooltip}
          entitlement={entitlement}
          addon={addon}
          addonPurchased={addonPurchased}
          onChange={onChange}
        />
      );
    } else {
      throw new Error('Boolean addons not supported yet');
    }
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
              <RiAlertFill className="-mt-0.5 inline h-4" /> Failed to update
              addon.{' '}
              <a href="/support" className="underline">
                Contact support
              </a>{' '}
              if this problem persists.
            </p>
          )}
          {entitlement.displayValue && !openSelfService ? (
            <div className="text-basis pr-3 text-sm font-medium">
              {entitlement.displayValue}
            </div>
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
        typeof entitlement.currentValue === 'number' && (
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
            'Are you sure you want to apply this change to your plan? ' +
            addonConfirmDescription
          }
          confirmButtonLabel={
            (addonCost || 0) > 0 ? 'Confirm and pay' : 'Confirm'
          }
          cancelButtonLabel="Cancel"
          confirmButtonKind="primary"
        />
      )}
    </>
  );
}
