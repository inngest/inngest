import { useState } from 'react';
import { Button } from '@inngest/components/Button/NewButton';
import { methodTypes } from '@inngest/components/types/app';
import { RiRefreshLine } from '@remixicon/react';

import ResyncModal from './ResyncModal';

type Props = {
  appExternalID: string;
  disabled?: boolean;
  latestSyncUrl: string;
  platform: string | null;
  appMethod: string;
};

export function ResyncButton({
  appExternalID,
  disabled = false,
  latestSyncUrl,
  platform,
  appMethod,
}: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button
        onClick={() => setIsModalVisible(true)}
        disabled={disabled}
        appearance={appMethod === methodTypes.Connect ? 'outlined' : 'solid'}
        kind="primary"
        label={appMethod === methodTypes.Connect ? 'Migrate' : 'Resync'}
        icon={<RiRefreshLine />}
        iconSide="left"
      />

      <ResyncModal
        appExternalID={appExternalID}
        appMethod={appMethod}
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
        url={latestSyncUrl}
        platform={platform}
      />
    </>
  );
}
