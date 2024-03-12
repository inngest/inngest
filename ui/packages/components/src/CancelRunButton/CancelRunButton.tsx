import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconStatusCanceled } from '@inngest/components/icons/status/Canceled';

import { CancelRunModal } from './CancelRunModal';

type Props = {
  disabled: boolean;
  onClick: () => Promise<unknown>;
};

export function CancelRunButton({ disabled, onClick }: Props) {
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
        icon={<IconStatusCanceled />}
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
