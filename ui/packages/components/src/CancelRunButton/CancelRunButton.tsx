import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconStatusCancelled } from '@inngest/components/icons/status/Cancelled';

import { CancelRunModal } from './CancelRunModal';

type Props = {
  disabled: boolean;

  /**
   * @deprecated Delete when we remove the old run details
   */
  hasIcon?: boolean;

  onClick: () => Promise<unknown>;
};

export function CancelRunButton({ disabled, hasIcon = false, onClick }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  async function onConfirm() {
    await onClick();
    setIsModalOpen(false);
  }

  return (
    <>
      <Button
        onClick={() => setIsModalOpen(true)}
        disabled={disabled}
        icon={hasIcon && <IconStatusCancelled />}
        iconSide="left"
        label="Cancel"
        size="medium"
        kind="secondary"
        appearance="outlined"
      />

      <CancelRunModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={onConfirm}
      />
    </>
  );
}
