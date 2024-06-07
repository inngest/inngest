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
      <Button
        appearance="outlined"
        btnAction={() => setIsModalVisible(true)}
        kind="primary"
        label="Validate"
      />

      <ValidateModal
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
        url={latestSyncUrl}
      />
    </>
  );
}
