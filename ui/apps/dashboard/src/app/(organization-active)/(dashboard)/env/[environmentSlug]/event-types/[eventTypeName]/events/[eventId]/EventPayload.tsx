'use client';

import { IconCloudArrowDown } from '@inngest/components/icons/CloudArrowDown';
import { devServerURL, useDevServer } from '@inngest/components/utils/useDevServer';
import { type JsonValue } from 'type-fest';

import DashboardCodeBlock from '@/components/DashboardCodeBlock/DashboardCodeBlock';
import { getFragmentData, graphql, type FragmentType } from '@/gql';

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
      header={{
        title: 'Payload',
      }}
      tab={{
        content: formattedPayload,
        language: 'json',
        readOnly: true,
      }}
      actions={[
        {
          label: 'Send to Dev Server',
          title: isRunning
            ? 'Send event payload to running Dev Server'
            : `Dev Server is not running at ${devServerURL}`,
          icon: <IconCloudArrowDown />,
          onClick: () => send(payload),
          disabled: !isRunning,
        },
      ]}
    />
  );
}
