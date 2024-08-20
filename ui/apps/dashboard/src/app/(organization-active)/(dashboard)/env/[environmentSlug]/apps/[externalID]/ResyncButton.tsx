import { useState } from 'react';
import { Button, NewButton } from '@inngest/components/Button';
import { RiRefreshLine } from '@remixicon/react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
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
      <NewButton
        onClick={() => setIsModalVisible(true)}
        disabled={disabled}
        kind="primary"
        label="Resync"
        icon={<RiRefreshLine />}
        iconSide="left"
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
