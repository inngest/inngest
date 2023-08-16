import { useState } from 'react';

import AddAppModal from '@/components/App/AddAppModal';
import Button from '@/components/Button';
import { IconPlus } from '@/icons';

export default function AddAppButton() {
  const [isAddAppModalVisible, setAddAppModalVisible] = useState(false);

  return (
    <>
      <Button
        label="Add App"
        icon={<IconPlus />}
        btnAction={() => setAddAppModalVisible(true)}
      />
      {isAddAppModalVisible && (
        <AddAppModal isOpen={isAddAppModalVisible} onClose={() => setAddAppModalVisible(false)} />
      )}
    </>
  );
}
