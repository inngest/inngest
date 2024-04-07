'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconTrash } from '@inngest/components/icons/Trash';

import { DeleteSigningKeyModal } from './DeleteSigningKeyModal';

type Props = {
  signingKeyID: string;
};

export function DeleteSigningKeyButton({ signingKeyID }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <Button
        appearance="outlined"
        aria-label="Delete"
        btnAction={() => setIsModalOpen(true)}
        icon={<IconTrash />}
        kind="danger"
        size="small"
        tooltip="Delete"
      />

      <DeleteSigningKeyModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        signingKeyID={signingKeyID}
      />
    </>
  );
}
