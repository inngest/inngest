import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import { ValidateModal } from './ValidateModal';

type Props = {
  latestSyncUrl: string;
};

export function ValidateButton({ latestSyncUrl }: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button btnAction={() => setIsModalVisible(true)} label="Start app diagnostic" />

      <ValidateModal
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
        url={latestSyncUrl}
      />
    </>
  );
}
