'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import { SendEventModal } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/events/[eventName]/SendEventModal';

type SendEventButtonProps = {
  eventName?: string;
};

export default function SendEventButton({ eventName }: SendEventButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button onClick={() => setIsModalVisible(true)} kind="primary" label="Send Event" />
      <SendEventModal
        isOpen={isModalVisible}
        eventName={eventName}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
