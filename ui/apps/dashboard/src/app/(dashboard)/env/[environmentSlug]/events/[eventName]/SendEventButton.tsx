'use client';

import { useState } from 'react';
import { PaperAirplaneIcon } from '@heroicons/react/20/solid';

import { SendEventModal } from '@/app/(dashboard)/env/[environmentSlug]/events/[eventName]/SendEventModal';
import Button from '@/components/Button';

type SendEventButtonProps = {
  environmentSlug: string;
  eventName?: string;
};

export default function SendEventButton({ environmentSlug, eventName }: SendEventButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button
        onClick={() => setIsModalVisible(true)}
        context="dark"
        icon={<PaperAirplaneIcon className="h-3" />}
      >
        Send Event
      </Button>
      <SendEventModal
        isOpen={isModalVisible}
        environmentSlug={environmentSlug}
        eventName={eventName}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
