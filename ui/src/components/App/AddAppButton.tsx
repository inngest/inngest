import { useState } from 'react';

import { IconPlus } from '@/icons';
import Button from '@/components/Button';
import AddAppModal from '@/components/App/AddAppModal';

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
        <AddAppModal
          isOpen={isAddAppModalVisible}
          onClose={() => setAddAppModalVisible(false)}
        />
      )}
    </>
  );
}
