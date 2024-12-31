'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';

import { useEnvironment } from '@/components/Environments/environment-context';
import { SendEventModal } from './SendEventModal';

type SendEventButtonProps = {
  eventName?: string;
};

export default function SendEventButton({ eventName }: SendEventButtonProps) {
  const { isArchived } = useEnvironment();

  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <OptionalTooltip tooltip={isArchived && 'Cannot send events. Environment is archived'}>
        <Button
          disabled={isArchived}
          onClick={() => setIsModalVisible(true)}
          kind="primary"
          label="Send event"
        />
      </OptionalTooltip>

      <SendEventModal
        isOpen={isModalVisible}
        eventName={eventName}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
