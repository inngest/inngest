import { useCallback, useMemo } from 'react';
import {
  SendEventModal as BaseSendEventModal,
  type SendEventConfig,
  type SharedSendEventModalProps,
} from '@inngest/components/SendEvent/SendEventModal';
import {
  generateSDKCode,
  generateCurlCode,
  type EventPayload,
} from '@inngest/components/SendEvent/utils';

import { usePortal } from '../../hooks/usePortal';
import { useSendEventMutation } from '../../store/devApi';
import { genericiseEvent } from '../../utils/events';

type DevServerSendEventModalProps = Omit<SharedSendEventModalProps, 'config'>;

export default function SendEventModal({
  data,
  isOpen,
  onClose,
}: DevServerSendEventModalProps) {
  const [_sendEvent, sendEventState] = useSendEventMutation();
  const portal = usePortal();

  const sendEvent = useCallback(
    async (payload: EventPayload | EventPayload[]) => {
      return _sendEvent(payload).unwrap();
    },
    [_sendEvent],
  );

  const portalWrapper = useCallback(
    (element: React.ReactElement) => portal(element) as React.ReactElement,
    [portal],
  );

  const processInitialData = useCallback(
    (data?: string | null) => genericiseEvent(data),
    [],
  );

  const config: SendEventConfig = useMemo(
    () => ({
      sendEvent,
      generateSDKCode: (payload) => generateSDKCode(payload),
      generateCurlCode: (payload) => generateCurlCode(payload),
      ui: {
        modalTitle: 'Send Event',
        sendButtonLabel: 'Send event',
        isLoading: sendEventState.isLoading,
      },
      usePortal: () => portalWrapper,
      processInitialData,
    }),
    [sendEvent, portalWrapper, sendEventState.isLoading, processInitialData],
  );

  return (
    <BaseSendEventModal
      data={data}
      isOpen={isOpen}
      onClose={onClose}
      config={config}
    />
  );
}
