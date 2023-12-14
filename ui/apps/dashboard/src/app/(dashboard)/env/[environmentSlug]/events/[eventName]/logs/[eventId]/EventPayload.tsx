'use client';

import { IconCloudArrowDown } from '@inngest/components/icons/CloudArrowDown';
import { type JsonValue } from 'type-fest';

import DashboardCodeBlock from '@/components/DashboardCodeBlock/DashboardCodeBlock';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { getFragmentData, graphql, type FragmentType } from '@/gql';
import { useDevServer } from '@/utils/useDevServer';

const EventPayloadFragment = graphql(`
  fragment EventPayload on ArchivedEvent {
    payload: event
  }
`);

type EventPayloadProps = {
  event: FragmentType<typeof EventPayloadFragment>;
};

export default function EventPayload({ event }: EventPayloadProps) {
  const { payload } = getFragmentData(EventPayloadFragment, event);
  const { isRunning, send } = useDevServer();
  const { value: isSendToDevServerEnabled } = useBooleanFlag('send-to-dev-server', false);

  let parsedPayload: string | JsonValue = '';
  if (typeof payload === 'string') {
    try {
      parsedPayload = JSON.parse(payload);
    } catch (error) {
      console.error(`Error parsing payload: `, error);
      parsedPayload = payload;
    }
  }

  const formattedPayload = JSON.stringify(parsedPayload, null, 2);

  return (
    <DashboardCodeBlock
      tabs={[
        {
          label: 'Payload',
          content: formattedPayload,
          language: 'json',
          readOnly: true,
        },
      ]}
      actions={
        isSendToDevServerEnabled
          ? [
              {
                label: 'Send to Dev Server',
                title: isRunning
                  ? 'Send event payload to running Dev Server'
                  : 'Dev Server is not running',
                icon: <IconCloudArrowDown />,
                onClick: () => send(payload),
                disabled: !isRunning,
              },
            ]
          : []
      }
    />
  );
}
