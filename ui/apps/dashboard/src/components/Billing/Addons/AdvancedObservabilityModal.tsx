import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@inngest/components/Button";
import { AlertModal } from "@inngest/components/Modal/AlertModal";
import { RiAlertFill } from "@remixicon/react";
import { toast } from "sonner";
import { useMutation } from "urql";

import { graphql } from "@/gql";

const UpdateAccountAddonQuantityDocument = graphql(`
  mutation UpdateAccountAddonQuantity($addonName: String!, $quantity: Int!) {
    updateAccountAddonQuantity(addonName: $addonName, quantity: $quantity) {
      purchaseCount
    }
  }
`);

export default function AdvancedObservabilityComponent({
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
    ? "Remove addon"
    : addonCost > 0
    ? "Confirm and pay"
    : "Confirm";

  const removingDescription = `Are you sure you want to remove the ${title.toLowerCase()} addon? You will revert back to your current plan's limits.`;

  const addingDescription = (
    <>
      By clicking Confirm and Pay, the amount of{" "}
      <span className="text-basis font-semibold">
        ${(addonCost / 100).toFixed(2)}
      </span>{" "}
      will be added to your subscription, and your credit card will be charged{" "}
      <span className="text-basis font-semibold">
        ${(addonCost / 100).toFixed(2)} immediately
      </span>{" "}
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
      toast.error("Failed to update addon. Please try again.");
    } else {
      setErr(null);
      if (onChange) {
        onChange();
      }
      router.refresh();
      toast.success(`Addon ${isRemoving ? "removed" : "updated"} successfully`);
    }
    setOpenModal(false);
  };

  return (
    <>
      {err && (
        <p className="text-error mb-2 text-xs">
          <RiAlertFill className="-mt-0.5 inline h-4" /> Failed to update addon.{" "}
          <a href="/support" className="underline">
            Contact support
          </a>{" "}
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
            label="Remove advanced observability"
            onClick={() => handleClick(true)}
          />
        ) : (
          <Button
            appearance="outlined"
            label="Add advanced observability"
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
          confirmButtonKind={isRemoving ? "danger" : "primary"}
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

              <div className="p-2">
                <div className="border-subtle pb-2">
                  <h3 className="text-basis text-md font-semibold">
                    Advanced Observability
                  </h3>
                </div>

                <div className="">
                  <div className="flex flex-col justify-between border-y py-2">
                    <span className="text-basis text-sm font-medium">
                      Log retention
                    </span>
                    <div className="flex items-center gap-2">
                      <span className="text-basis text-sm font-medium">
                        30 days
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-col justify-between border-b py-3">
                    <span className="text-basis text-sm font-medium">
                      Metrics granularity
                    </span>
                    <div className="flex items-center gap-2">
                      <span className="text-basis text-sm font-medium">
                        1 minute
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-col justify-between border-b py-3">
                    <span className="text-basis text-sm font-medium">
                      Metrics freshness
                    </span>
                    <div className="flex items-center gap-2">
                      <span className="text-basis text-sm font-medium">
                        5 minutes
                      </span>
                    </div>
                  </div>
                </div>

                <div className="border-subtle flex items-center justify-between border-t pt-4">
                  <span className="text-basis">Add on cost</span>
                  <span className="text-basis">
                    ${(addonCost / 100).toFixed(2)}/mo
                  </span>
                </div>
              </div>
            </div>
          )}
        </AlertModal>
      )}
    </>
  );
}
