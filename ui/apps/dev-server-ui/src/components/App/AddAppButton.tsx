import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiAddLine } from '@remixicon/react';

import AddAppModal from '@/components/App/AddAppModal';

export default function AddAppButton({ secondary }: { secondary?: boolean }) {
  const [isAddAppModalVisible, setAddAppModalVisible] = useState(false);

  return (
    <>
      <Button
        kind={secondary ? 'secondary' : 'primary'}
        label={secondary ? 'I want to sync manually' : 'Sync new app'}
        appearance={secondary ? 'outlined' : 'solid'}
        icon={secondary ? null : <RiAddLine />}
        iconSide="left"
        onClick={() => setAddAppModalVisible(true)}
      />

      {isAddAppModalVisible && (
        <AddAppModal isOpen={isAddAppModalVisible} onClose={() => setAddAppModalVisible(false)} />
      )}
    </>
  );
}
