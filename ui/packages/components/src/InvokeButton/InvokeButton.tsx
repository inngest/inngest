import { useCallback, useState } from 'react';
import { Button, type ButtonKind } from '@inngest/components/Button';
import type { ButtonAppearance } from '@inngest/components/Button/Button';
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

  // appearance = 'solid',
  return (
    <>
      <Button
        kind={kind}
        appearance={appearance}
        onClick={() => setIsModalOpen(true)}
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
