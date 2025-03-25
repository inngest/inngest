import { useCallback, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { InvokeModal } from '@inngest/components/InvokeButton';
import { RiFlashlightFill } from '@remixicon/react';

type Props = {
  disabled?: boolean;
  doesFunctionAcceptPayload: boolean;
  btnAction: (payload: {
    data: Record<string, unknown>;
    user: Record<string, unknown> | null;
  }) => void;
};

export function InvokeButton({
  disabled,
  doesFunctionAcceptPayload: hasEventTrigger,
  btnAction,
}: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  const onConfirm = useCallback(
    (payload: { data: Record<string, unknown>; user: Record<string, unknown> | null }) => {
      setIsModalOpen(false);
      btnAction(payload);
    },
    [setIsModalOpen, btnAction]
  );

  return (
    <>
      <Button
        kind="secondary"
        appearance="outlined"
        onClick={() => setIsModalOpen(true)}
        disabled={disabled}
        icon={<RiFlashlightFill className="text-sky-500" />}
        iconSide="left"
        label="Invoke"
      />

      <InvokeModal
        doesFunctionAcceptPayload={hasEventTrigger}
        isOpen={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onConfirm={onConfirm}
      />
    </>
  );
}
