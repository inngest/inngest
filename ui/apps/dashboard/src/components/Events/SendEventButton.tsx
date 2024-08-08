'use client';

import { useState } from 'react';
import { Button, NewButton } from '@inngest/components/Button';

import { useEnvironment } from '@/components/Environments/environment-context';
import { OptionalTooltip } from '@/components/Navigation/OptionalTooltip';
import { SendEventModal } from './SendEventModal';

type SendEventButtonProps = {
  eventName?: string;
  newIANav?: boolean;
};

export default function SendEventButton({ eventName, newIANav = false }: SendEventButtonProps) {
  const { isArchived } = useEnvironment();

  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <OptionalTooltip tooltip={isArchived && 'Cannot send events. Environment is archived'}>
        {newIANav ? (
          <NewButton
            disabled={isArchived}
            onClick={() => setIsModalVisible(true)}
            kind="primary"
            label="Send Event"
          />
        ) : (
          <Button
            disabled={isArchived}
            btnAction={() => setIsModalVisible(true)}
            kind="primary"
            label="Send Event"
          />
        )}
      </OptionalTooltip>

      <SendEventModal
        isOpen={isModalVisible}
        eventName={eventName}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
