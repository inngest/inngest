import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiAddLine } from '@remixicon/react';

import AddAppModal from '@/components/App/AddAppModal';

export default function AddAppButton() {
  const [isAddAppModalVisible, setAddAppModalVisible] = useState(false);

  return (
    <>
      <Button
        label="Sync New App"
        icon={<RiAddLine />}
        btnAction={() => setAddAppModalVisible(true)}
      />
      {isAddAppModalVisible && (
        <AddAppModal isOpen={isAddAppModalVisible} onClose={() => setAddAppModalVisible(false)} />
      )}
    </>
  );
}
