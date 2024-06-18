import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import { ArchiveModal } from './ArchiveModal';

type Props = {
  appID: string;
  disabled?: boolean;
  isArchived: boolean;
};

export function ArchiveButton({ appID, disabled = false, isArchived }: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  let label = 'Archive';
  if (isArchived) {
    label = 'Unarchive';
  }

  return (
    <>
      <Button btnAction={() => setIsModalVisible(true)} disabled={disabled} label={label} />

      <ArchiveModal
        appID={appID}
        isArchived={isArchived}
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
