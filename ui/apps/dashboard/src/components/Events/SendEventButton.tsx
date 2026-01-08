import { useState, lazy, Suspense } from 'react';
import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';

import { useEnvironment } from '@/components/Environments/environment-context';

const SendEventModal = lazy(() =>
  import('./SendEventModal').then((m) => ({ default: m.SendEventModal })),
);

type SendEventButtonProps = {
  eventName?: string;
};

export default function SendEventButton({ eventName }: SendEventButtonProps) {
  const { isArchived } = useEnvironment();

  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <OptionalTooltip
        tooltip={isArchived && 'Cannot send events. Environment is archived'}
      >
        <Button
          disabled={isArchived}
          onClick={() => setIsModalVisible(true)}
          kind="primary"
          label="Send event"
        />
      </OptionalTooltip>

      {isModalVisible && (
        <Suspense fallback={null}>
          <SendEventModal
            isOpen={isModalVisible}
            eventName={eventName}
            onClose={() => setIsModalVisible(false)}
          />
        </Suspense>
      )}
    </>
  );
}
