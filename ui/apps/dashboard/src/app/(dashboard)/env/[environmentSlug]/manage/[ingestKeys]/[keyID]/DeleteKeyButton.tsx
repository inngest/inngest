'use client';

import { useState } from 'react';
import { TrashIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import useManagePageTerminology from './../useManagePageTerminology';
import DeleteKeyModal from './DeleteKeyModal';

type ArchiveKeyProps = {
  environmentSlug: string;
  environmentID: string;
  keyID: string;
};

export default function DeleteKeyButton({
  environmentSlug,
  environmentID,
  keyID,
}: ArchiveKeyProps) {
  const [isDeleteKeyModalVisible, setIsDeleteKeyModalVisible] = useState<boolean>(false);
  const currentContent = useManagePageTerminology();

  return (
    <>
      <Button
        icon={<TrashIcon />}
        appearance="outlined"
        kind="danger"
        btnAction={() => setIsDeleteKeyModalVisible(true)}
        label={'Delete ' + currentContent?.titleType}
      />
      {isDeleteKeyModalVisible && (
        <DeleteKeyModal
          environmentID={environmentID}
          environmentSlug={environmentSlug}
          keyID={keyID}
          isOpen={isDeleteKeyModalVisible}
          onClose={() => setIsDeleteKeyModalVisible(false)}
        />
      )}
    </>
  );
}
