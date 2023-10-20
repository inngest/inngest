'use client';

import { useState } from 'react';
import { PaperAirplaneIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import { SendEventModal } from '@/app/(dashboard)/env/[environmentSlug]/events/[eventName]/SendEventModal';

type SendEventButtonProps = {
  environmentSlug: string;
  eventName?: string;
};

export default function SendEventButton({ environmentSlug, eventName }: SendEventButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button
        btnAction={() => setIsModalVisible(true)}
        kind="primary"
        icon={<PaperAirplaneIcon />}
        label="Send Event"
      />
      <SendEventModal
        isOpen={isModalVisible}
        environmentSlug={environmentSlug}
        eventName={eventName}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
