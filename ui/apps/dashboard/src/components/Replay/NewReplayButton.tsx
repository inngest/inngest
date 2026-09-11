import { useState } from 'react';

import NewReplayModal from '@/components/Replay/NewReplayModal';

type NewReplayButtonProps = {
  functionID: string;
  functionSlug: string;
};

export default function NewReplayButton({
  functionID,
  functionSlug,
}: NewReplayButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <NewReplayModal
      isOpen={isModalVisible}
      functionID={functionID}
      functionSlug={functionSlug}
      onClose={() => setIsModalVisible(false)}
    />
  );
}
