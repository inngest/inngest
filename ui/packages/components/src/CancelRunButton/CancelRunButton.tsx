import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconStatusCancelled } from '@inngest/components/icons/status/Cancelled';

import { useCancelRun } from '../SharedContext/useCancelRun';
import { CancelRunModal } from './CancelRunModal';

type Props = {
  disabled: boolean;

  /**
   * @deprecated Delete when we remove the old run details
   */
  hasIcon?: boolean;

  runID: string;
};

export function CancelRunButton({ disabled, hasIcon = false, runID }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const { cancelRun } = useCancelRun();

  async function onConfirm() {
    await cancelRun({ runID });
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
