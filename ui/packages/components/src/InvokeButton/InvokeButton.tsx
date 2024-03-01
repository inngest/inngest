import { useCallback, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { InvokeModal } from '@inngest/components/InvokeButton';
import { IconTriggerFunction } from '@inngest/components/icons/TriggerFunction';

type Props = {
  disabled?: boolean;
  doesFunctionAcceptPayload: boolean;
  btnAction: (data: Record<string, unknown>) => void;
};

export function InvokeButton({
  disabled,
  doesFunctionAcceptPayload: hasEventTrigger,
  btnAction,
}: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  const onConfirm = useCallback(
    ({ data }: { data: Record<string, unknown> }) => {
      console.log('onConfirm hit!!');
      setIsModalOpen(false);
      btnAction(data);
    },
    [setIsModalOpen, btnAction]
  );

  return (
    <>
      <Button
        btnAction={() => setIsModalOpen(true)}
        disabled={disabled}
        icon={<IconTriggerFunction />}
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
