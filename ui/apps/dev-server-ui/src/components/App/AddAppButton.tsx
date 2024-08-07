import { useState } from 'react';
import { Button, NewButton } from '@inngest/components/Button';
import { RiAddLine } from '@remixicon/react';

import AddAppModal from '@/components/App/AddAppModal';

export default function AddAppButton() {
  const [isAddAppModalVisible, setAddAppModalVisible] = useState(false);

  return (
    <>
      <NewButton
        kind="primary"
        label="Sync new app"
        icon={<RiAddLine />}
        iconSide="left"
        onClick={() => setAddAppModalVisible(true)}
      />

      {isAddAppModalVisible && (
        <AddAppModal isOpen={isAddAppModalVisible} onClose={() => setAddAppModalVisible(false)} />
      )}
    </>
  );
}
