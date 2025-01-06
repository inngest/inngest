'use client';

import { useCallback, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal/Modal';
import TabCards from '@inngest/components/TabCards/TabCards';
import ky from 'ky';
import { toast } from 'sonner';
import { type JsonValue } from 'type-fest';
import { useQuery } from 'urql';
import { z } from 'zod';

import { useEnvironment } from '@/components/Environments/environment-context';
import CodeEditor from '@/components/Textarea/CodeEditor';
import { graphql } from '@/gql';
import { EnvironmentType } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';

const eventSchema = z.object({
  name: z.string(),
  data: z.record(z.unknown()),
});

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

type SendEventModalProps = {
  eventName?: string;
  isOpen: boolean;
  onClose: () => void;
};

type TabType = {
  payload: { name: string; data: {} } | { name: string; data: {} }[];
  eventKey?: string;
  isBranchChild: boolean;
  envName: string;
  sendEventURL: string;
  sendEventAction: (event: React.FormEvent<HTMLFormElement>) => void;
  copyToClipboardAction: (event: React.FormEvent<HTMLFormElement>) => void;
};

const buildTabs = ({
  payload,
  eventKey,
  envName,
  sendEventURL,
  isBranchChild,
  sendEventAction,
  copyToClipboardAction,
}: TabType) => {
  return [
    {
      tabLabel: 'JSON Editor',
      tabTitle: 'Send Custom JSON',
      submitButtonLabel: 'Send event',
      submitButtonEnabled: Boolean(eventKey),
      submitAction: sendEventAction,
      codeLanguage: 'json',
      initialCode: JSON.stringify(payload, null, 2),
    },
    {
      tabLabel: 'SDK',
      tabTitle: 'Send with the SDK',
      submitButtonLabel: 'Copy Code',
      submitButtonEnabled: true,
      submitAction: copyToClipboardAction,
      codeLanguage: 'javascript',
      initialCode: `import { Inngest } from 'inngest';

const inngest = new Inngest({
  name: 'Your App Name',
  eventKey: '${eventKey || '<EVENT_KEY>'}',${isBranchChild ? `\n  env: '${envName}',` : ''}
});

await inngest.send(${JSON.stringify(payload, null, 2)});`,
    },
    {
      tabLabel: 'cURL',
      tabTitle: 'Send with cURL',
      submitButtonLabel: 'Copy Code',
      submitButtonEnabled: true,
      submitAction: copyToClipboardAction,
      codeLanguage: 'bash',
      initialCode: `curl ${sendEventURL} \\${
        isBranchChild ? `\n  -H "x-inngest-env: ${envName}" \\` : ''
      }
  --data '${JSON.stringify(payload)}'`,
    },
  ];
};

export function SendEventModal({
  eventName = 'Your Event Name',
  isOpen,
  onClose,
}: SendEventModalProps) {
  const [payload, setPayload] = useState<
    z.infer<typeof eventSchema> | z.infer<typeof eventSchema>[]
  >({
    name: eventName,
    data: {},
  });
  const router = useRouter();
  const environment = useEnvironment();
  const eventKey = usePreferDefaultEventKey();
  const hasEventKey = Boolean(eventKey);
  const protocol = process.env.NODE_ENV === 'development' ? 'http://' : 'https://';
  const sendEventURL = `${protocol}${process.env.NEXT_PUBLIC_EVENT_API_HOST}/e/${
    eventKey || '<EVENT_KEY>'
  }`;

  const isBranchChild = environment.type === EnvironmentType.BranchChild;
  const envName = environment.name;

  // serialize data to state on change so we can persist it between editor tab changes
  const serializeData = (code: string) => {
    let payload;
    const parsedObject = eventSchema.safeParse(JSON.parse(code));
    if (parsedObject.success) {
      payload = parsedObject.data;
    } else {
      const parsedArray = z.array(eventSchema).safeParse(JSON.parse(code));
      if (parsedArray.success) {
        payload = parsedArray.data;
      } else {
        console.log("can't parse code editor payload, skipping serialization");
        return;
      }
    }

    setPayload(payload);
  };

  const sendEventAction = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = event.currentTarget;
      const formData = new FormData(form);
      const jsonString = formData.get('code') as string;

      let jsonEvent: JsonValue;
      try {
        jsonEvent = JSON.parse(jsonString);
      } catch (error) {
        toast.error('Could not parse JSON. Please check your syntax.');
        return;
      }

      const headers: { ['x-inngest-env']?: string } = {};
      if (isBranchChild) {
        headers['x-inngest-env'] = envName;
      }

      const sendEvent = ky.post(sendEventURL, {
        json: jsonEvent,
        headers,
      });

      toast.promise(sendEvent, {
        loading: 'Loading...',
        success: () => {
          router.refresh();
          onClose();
          window.location.reload(); // We need to reload page to display new events, because we can't update the URQL cache without using mutations
          return 'Event sent!';
        },
        error: 'Could not send event. Please try again later.',
      });
    },
    [envName, isBranchChild, onClose, router, sendEventURL]
  );

  const copyToClipboardAction = useCallback(
    (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = event.currentTarget;
      const formData = new FormData(form);
      const code = formData.get('code') as string;

      toast.promise(navigator.clipboard.writeText(code), {
        loading: 'Loading...',
        success: () => {
          router.refresh();
          onClose();
          return 'Copied to clipboard!';
        },
        error: 'Could not copy to clipboard.',
      });
    },
    [onClose, router]
  );

  let tabs = useMemo(() => {
    return buildTabs({
      envName,
      payload,
      eventKey,
      sendEventURL,
      isBranchChild,
      copyToClipboardAction,
      sendEventAction,
    });
  }, [
    copyToClipboardAction,
    envName,
    eventKey,
    isBranchChild,
    payload,
    sendEventAction,
    sendEventURL,
  ]);

  return (
    <Modal className="max-w-6xl" isOpen={isOpen} onClose={onClose}>
      <Modal.Body>
        <TabCards
          defaultValue="JSON Editor"
          onChange={() => {
            tabs = buildTabs({
              envName,
              payload,
              eventKey,
              sendEventURL,
              isBranchChild,
              copyToClipboardAction,
              sendEventAction,
            });
          }}
        >
          <div className="items-top flex justify-between">
            <h2 className="text-basis text-xl">Send Event</h2>
            <TabCards.ButtonList>
              {tabs.map(({ tabLabel }) => (
                <TabCards.Button key={tabLabel} value={tabLabel}>
                  {tabLabel}
                </TabCards.Button>
              ))}
            </TabCards.ButtonList>
          </div>
          {!hasEventKey && (
            <Alert severity="warning" className="mb-2 text-sm">
              There are no Event Keys for this environment. Please create an Event Key in{' '}
              <Alert.Link href={pathCreator.keys({ envSlug: environment.slug })} severity="warning">
                the Manage tab
              </Alert.Link>{' '}
              first.
            </Alert>
          )}
          <>
            {tabs.map(
              ({
                tabLabel,
                tabTitle,
                submitButtonLabel,
                submitButtonEnabled,
                submitAction,
                codeLanguage,
                initialCode,
              }) => (
                <TabCards.Content key={tabLabel} value={tabLabel} className="p-0">
                  <form onSubmit={submitAction}>
                    <header className="flex items-center justify-between rounded-t-md p-2">
                      <h3 className="px-2">{tabTitle}</h3>
                      <Button
                        type="submit"
                        disabled={!submitButtonEnabled}
                        label={submitButtonLabel}
                        kind="primary"
                      />
                    </header>
                    <div className="bg-codeEditor w-full overflow-auto rounded-b-md p-4">
                      <CodeEditor
                        language={codeLanguage}
                        initialCode={initialCode}
                        name="code"
                        className="h-80 w-[640px]"
                        onCodeChange={serializeData}
                      />
                    </div>
                  </form>
                </TabCards.Content>
              )
            )}
          </>
        </TabCards>
      </Modal.Body>
      <Modal.Footer className="flex justify-end gap-2">
        <Button kind="secondary" label="Close modal" appearance="outlined" onClick={onClose} />
      </Modal.Footer>
    </Modal>
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
