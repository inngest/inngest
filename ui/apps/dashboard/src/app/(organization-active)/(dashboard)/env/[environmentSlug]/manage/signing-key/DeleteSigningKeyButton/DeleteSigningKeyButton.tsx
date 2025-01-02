'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiDeleteBin2Line } from '@remixicon/react';

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
        onClick={() => setIsModalOpen(true)}
        icon={<RiDeleteBin2Line />}
        kind="danger"
        size="small"
      />

      <DeleteSigningKeyModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        signingKeyID={signingKeyID}
      />
    </>
  );
}
