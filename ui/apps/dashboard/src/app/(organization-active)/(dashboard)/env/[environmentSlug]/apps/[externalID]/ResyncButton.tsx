import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import ResyncModal from './ResyncModal';

type Props = {
  appExternalID: string;
  latestSyncUrl: string;
  platform: string | null;
};

export function ResyncButton({ appExternalID, latestSyncUrl, platform }: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button btnAction={() => setIsModalVisible(true)} kind="primary" label="Resync" />

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
