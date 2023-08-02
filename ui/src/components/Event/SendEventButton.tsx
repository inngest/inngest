import { useState } from 'react';

import Button from '@/components/Button';
import SendEventModal from '@/components/Event/SendEventModal';

type SendEventButtonProps = {
  data?: string | null;
  label: string;
  kind?: 'primary' | 'secondary' | 'text';
};

export default function SendEventButton({ data, label, kind }: SendEventButtonProps) {
  const [isSendEventModalVisible, setSendEventModalVisible] = useState(false);

  return (
    <>
      <Button label={label} kind={kind} btnAction={() => setSendEventModalVisible(true)} />
      {isSendEventModalVisible && (
        <SendEventModal
          data={data}
          isOpen={isSendEventModalVisible}
          onClose={() => setSendEventModalVisible(false)}
        />
      )}
    </>
  );
}
