import { useState, type ReactNode } from 'react';

import { Button } from '@inngest/components/Button/NewButton';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { RiAlertFill } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { useNavigate, useRouter } from '@tanstack/react-router';

const UpdateAccountAddonQuantityDocument = graphql(`
  mutation UpdateAccountAddonQuantity($addonName: String!, $quantity: Int!) {
    updateAccountAddonQuantity(addonName: $addonName, quantity: $quantity) {
      purchaseCount
    }
  }
`);

export default function SlackChannelComponent({
  title,
  description,
  tooltip,
  entitlement,
  addon,
  addonPurchased,
  onChange,
}: {
  title: string;
  description: string | ReactNode;
  tooltip?: ReactNode;
  entitlement: {
    currentValue: number | boolean;
    displayValue?: string | ReactNode;
  };
  addon: {
    addonName: string;
    price: number;
  };
  addonPurchased?: boolean;
  onChange?: () => void;
}) {
  const router = useRouter();
  const [, updateAccountAddonQuantity] = useMutation(
    UpdateAccountAddonQuantityDocument,
  );
  const [openModal, setOpenModal] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const addonQty = isRemoving ? 0 : 1;
  const addonCost = addonQty * addon.price;

  const modalTitle = isRemoving
    ? `Remove ${title} addon`
    : `Add ${title.toLowerCase()} to plan`;

  const confirmButtonLabel = isRemoving
    ? 'Remove addon'
    : addonCost > 0
    ? 'Confirm and pay'
    : 'Confirm';

  const removingDescription = `Are you sure you want to remove the ${title.toLowerCase()} addon? You will lose access to the dedicated Slack channel for support.`;

  const addingDescription = (
    <>
      By clicking Confirm and Pay, the amount of{' '}
      <span className="text-basis font-semibold">
        ${(addonCost / 100).toFixed(2)}
      </span>{' '}
      will be added to your subscription, and your credit card will be charged{' '}
      <span className="text-basis font-semibold">
        ${(addonCost / 100).toFixed(2)} immediately
      </span>{' '}
      for the remaining days in your billing cycle.
    </>
  );

  const handleClick = (isRemoval: boolean) => {
    setIsRemoving(isRemoval);
    setOpenModal(true);
  };

  const handleSubmit = async () => {
    const updateResult = await updateAccountAddonQuantity({
      addonName: addon.addonName,
      quantity: addonQty,
    });
    if (updateResult.error) {
      console.error(updateResult.error.message);
      setErr(updateResult.error.message);
      toast.error('Failed to update addon. Please try again.');
    } else {
      setErr(null);
      if (onChange) {
        onChange();
      }
      router.invalidate();
      toast.success(`Addon ${isRemoving ? 'removed' : 'updated'} successfully`);
    }
    setOpenModal(false);
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
      <div className="flex items-end justify-between">
        <div>
          <p className="text-subtle mb-0.5 flex items-center gap-1 text-sm font-medium">
            {title}
            {tooltip && tooltip}
          </p>
          <p className="text-muted mb-1 text-sm italic">{description}</p>
          {entitlement.displayValue && (
            <div className="text-basis pr-3 text-sm font-medium">
              {entitlement.displayValue}
            </div>
          )}
        </div>
        {addonPurchased ? (
          <Button
            appearance="outlined"
            label="Remove Slack channel"
            onClick={() => handleClick(true)}
          />
        ) : (
          <Button
            appearance="outlined"
            label="Add Slack channel"
            onClick={() => handleClick(false)}
          />
        )}
      </div>
      {openModal && (
        <AlertModal
          isOpen={openModal}
          onClose={() => setOpenModal(false)}
          onSubmit={handleSubmit}
          title={modalTitle}
          confirmButtonLabel={confirmButtonLabel}
          cancelButtonLabel="Cancel"
          confirmButtonKind={isRemoving ? 'danger' : 'primary'}
          className="w-full max-w-lg"
        >
          {isRemoving ? (
            <div className="space-y-2 p-6">
              <p className="text-muted text-sm leading-relaxed">
                {removingDescription}
              </p>
            </div>
          ) : (
            <div className="space-y-2 p-6">
              <p className="text-muted text-sm leading-relaxed">
                {addingDescription}
              </p>
            </div>
          )}
        </AlertModal>
      )}
    </>
  );
}
