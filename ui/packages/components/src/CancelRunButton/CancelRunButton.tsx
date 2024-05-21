import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconStatusCanceled } from '@inngest/components/icons/status/Canceled';

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
        btnAction={() => setIsModalOpen(true)}
        disabled={disabled}
        icon={hasIcon && <IconStatusCanceled />}
        label="Cancel"
        size="small"
      />

      <CancelRunModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={onConfirm}
      />
    </>
  );
}
