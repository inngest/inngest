import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import AddAppModal from '@/components/App/AddAppModal';
import { IconPlus } from '@/icons';

export default function AddAppButton() {
  const [isAddAppModalVisible, setAddAppModalVisible] = useState(false);

  return (
    <>
      <Button
        label="Sync New App"
        icon={<IconPlus />}
        btnAction={() => setAddAppModalVisible(true)}
      />
      {isAddAppModalVisible && (
        <AddAppModal isOpen={isAddAppModalVisible} onClose={() => setAddAppModalVisible(false)} />
      )}
    </>
  );
}
