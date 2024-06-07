import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import SendEventModal from '@/components/Event/SendEventModal';

type SendEventButtonProps = {
  data?: string | null;
  label: string;
  appearance?: 'solid' | 'outlined' | 'ghost';
};

export default function SendEventButton({ data, label, appearance }: SendEventButtonProps) {
  const [isSendEventModalVisible, setSendEventModalVisible] = useState(false);

  return (
    <>
      <Button
        label={label}
        appearance={appearance}
        onClick={() => setSendEventModalVisible(true)}
      />
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
