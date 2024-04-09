'use client';

import React, { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconStatusCanceled } from '@inngest/components/icons/status/Canceled';

import { CreateCancellationModal } from './CreateCancellationModal';

type Props = {
  onSubmit: React.ComponentProps<typeof CreateCancellationModal>['onSubmit'];
};

export function CreateCancellationButton({ onSubmit }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <Button
        btnAction={() => setIsModalOpen(true)}
        icon={<IconStatusCanceled className="h-5 w-5 text-white" />}
        kind="primary"
        label="New Cancellation"
      />
      <CreateCancellationModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={async (data) => {
          await onSubmit(data);
          setIsModalOpen(false);
        }}
      />
    </>
  );
}
