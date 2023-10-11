'use client';

import { useState } from 'react';
import { TrashIcon } from '@heroicons/react/20/solid';

import Button from '@/components/Button';
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
        icon={<TrashIcon className="h-4" />}
        variant="secondary"
        onClick={() => setIsDeleteKeyModalVisible(true)}
        className="text-red-500 hover:text-red-700"
      >
        {'Delete ' + currentContent?.titleType}
      </Button>
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
