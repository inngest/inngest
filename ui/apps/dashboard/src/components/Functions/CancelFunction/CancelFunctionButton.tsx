import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import { CancelFunctionModal } from './CancelFunctionModal';

type Props = {
  envID: string;
  functionSlug: string;
};

export function CancelFunctionButton({ envID, functionSlug }: Props) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button btnAction={() => setIsModalVisible(true)} label="Bulk cancel" />

      <CancelFunctionModal
        envID={envID}
        functionSlug={functionSlug}
        isOpen={isModalVisible}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
