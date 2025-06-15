import { useCallback, useState } from 'react';
import { Button, type ButtonAppearance, type ButtonKind } from '@inngest/components/Button';
import { InvokeModal } from '@inngest/components/InvokeButton';

type Props = {
  kind?: ButtonKind;
  appearance?: ButtonAppearance;
  disabled?: boolean;
  doesFunctionAcceptPayload: boolean;
  btnAction: (payload: {
    data: Record<string, unknown>;
    user: Record<string, unknown> | null;
  }) => void;
};

export function InvokeButton({
  kind,
  appearance,
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
        kind={kind}
        appearance={appearance}
        onClick={(e) => {
          e.stopPropagation();
          setIsModalOpen(true);
        }}
        disabled={disabled}
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
