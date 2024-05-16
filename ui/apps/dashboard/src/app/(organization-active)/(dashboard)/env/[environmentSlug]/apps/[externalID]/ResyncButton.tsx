import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import ResyncModal from './ResyncModal';

type Props = {
  appExternalID: string;
  disabled?: boolean;
  latestSyncUrl: string;
  platform: string | null;
};

export function ResyncButton({ appExternalID, disabled = false, latestSyncUrl, platform }: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button
        btnAction={() => setIsModalVisible(true)}
        disabled={disabled}
        kind="primary"
        label="Resync"
      />

      <ResyncModal
        appExternalID={appExternalID}
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
        url={latestSyncUrl}
        platform={platform}
      />
    </>
  );
}
