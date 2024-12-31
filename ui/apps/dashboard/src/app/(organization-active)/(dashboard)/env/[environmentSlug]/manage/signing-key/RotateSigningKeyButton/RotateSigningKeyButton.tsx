'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import { RotateSigningKeyModal } from './RotateSigningKeyModal';

type Props = {
  disabled?: boolean;
  envID: string;
};

export function RotateSigningKeyButton({ disabled, envID }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <Button
        onClick={() => setIsModalOpen(true)}
        disabled={disabled}
        kind="danger"
        label="Rotate key"
      />

      <RotateSigningKeyModal
        envID={envID}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </>
  );
}
