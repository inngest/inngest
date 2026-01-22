import { useCallback, useMemo } from 'react';
import {
  SendEventModal as BaseSendEventModal,
  type SendEventConfig,
  type SharedSendEventModalProps,
} from '@inngest/components/SendEvent/SendEventModal';
import { type EventPayload } from '@inngest/components/SendEvent/utils';
import ky from 'ky';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { EnvironmentType } from '@/gql/graphql';
import { useRouter } from '@tanstack/react-router';

const GetEventKeysDocument = graphql(/* GraphQL */ `
  query GetEventKeys($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      eventKeys: ingestKeys {
        name
        value: presharedKey
      }
    }
  }
`);

type CloudSendEventModalProps = Omit<SharedSendEventModalProps, 'config'> & {
  eventName?: string;
  initialData?: string;
};

export function SendEventModal({
  data,
  isOpen,
  onClose,
  eventName = 'Your Event Name',
  initialData,
}: CloudSendEventModalProps) {
  const router = useRouter();
  const environment = useEnvironment();
  const eventKey = usePreferDefaultEventKey();

  const protocol =
    import.meta.env.MODE === 'development' ? 'http://' : 'https://';
  const sendEventURL = `${protocol}${import.meta.env.VITE_EVENT_API_HOST}/e/${
    eventKey || '<EVENT_KEY>'
  }`;

  const isBranchChild = environment.type === EnvironmentType.BranchChild;
  const envName = environment.name;

  const sendEvent = useCallback(
    async (payload: EventPayload | EventPayload[]) => {
      if (!eventKey) {
        throw new Error(
          'No event key available. Please check your environment configuration.',
        );
      }

      const headers: { ['x-inngest-env']?: string } = {};
      if (isBranchChild) {
        headers['x-inngest-env'] = envName;
      }

      await ky.post(sendEventURL, {
        json: payload,
        headers,
      });

      router.invalidate();
      window.location.reload(); // We need to reload page to display new events, because we can't update the URQL cache without using mutations
    },
    [sendEventURL, isBranchChild, envName, router, eventKey],
  );

  const generateCloudSDKCode = useCallback(
    (payload: EventPayload | EventPayload[]) => {
      return `import { Inngest } from 'inngest';

const inngest = new Inngest({
  name: 'Your App Name',
  eventKey: '${eventKey || '<EVENT_KEY>'}',${
        isBranchChild ? `\n  env: '${envName}',` : ''
      }
});

await inngest.send(${JSON.stringify(payload, null, 2)});`;
    },
    [eventKey, isBranchChild, envName],
  );

  const generateCloudCurlCode = useCallback(
    (payload: EventPayload | EventPayload[]) => {
      return `curl ${sendEventURL} \\${
        isBranchChild ? `\n  -H "x-inngest-env: ${envName}" \\` : ''
      }
  --data '${JSON.stringify(payload)}'`;
    },
    [sendEventURL, isBranchChild, envName],
  );

  const processCloudData = useCallback(() => {
    try {
      if (initialData) {
        const parsedData = JSON.parse(initialData);
        if (Array.isArray(parsedData)) {
          return JSON.stringify(
            parsedData.map((item) => ({
              name: item.name || eventName,
              data: item.data || {},
            })),
            null,
            2,
          );
        } else {
          return JSON.stringify(
            {
              name: eventName,
              data: parsedData.data || {},
            },
            null,
            2,
          );
        }
      }
      return JSON.stringify({ name: eventName, data: {} }, null, 2);
    } catch (error) {
      console.error('Failed to parse initialData:', error);
      return JSON.stringify({ name: eventName, data: {} }, null, 2);
    }
  }, [eventName, initialData]);

  const config: SendEventConfig = useMemo(
    () => ({
      sendEvent,
      generateSDKCode: generateCloudSDKCode,
      generateCurlCode: generateCloudCurlCode,
      ui: {
        modalTitle: 'Send Event',
        sendButtonLabel: 'Send event',
        isLoading: false,
      },
      processInitialData: processCloudData,
    }),
    [sendEvent, generateCloudSDKCode, generateCloudCurlCode, processCloudData],
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

function usePreferDefaultEventKey(): string | undefined {
  const environment = useEnvironment();
  const [{ data }] = useQuery({
    query: GetEventKeysDocument,
    variables: {
      environmentID: environment.id,
    },
  });

  const eventKeys = data?.environment.eventKeys;

  const defaultKey = eventKeys?.find((eventKey) => {
    return eventKey.name?.toLowerCase().startsWith('default in');
  })?.value;

  return defaultKey ?? eventKeys?.[0]?.value;
}
