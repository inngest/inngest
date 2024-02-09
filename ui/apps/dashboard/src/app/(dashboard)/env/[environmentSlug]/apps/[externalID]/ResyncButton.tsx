import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import ResyncModal from './ResyncModal';

type Props = {
  latestSyncUrl: string;
  platform: string;
};

export function ResyncButton({ latestSyncUrl, platform }: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button btnAction={() => setIsModalVisible(true)} kind="primary" label="Resync" />

      <ResyncModal
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
        url={latestSyncUrl}
        platform={platform}
      />
    </>
  );
}
