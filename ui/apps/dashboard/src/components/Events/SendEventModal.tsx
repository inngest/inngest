'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Tab } from '@headlessui/react';
import { Alert } from '@inngest/components/Alert';
import { NewButton } from '@inngest/components/Button';
import ky from 'ky';
import { toast } from 'sonner';
import { type JsonValue } from 'type-fest';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import Modal from '@/components/Modal';
import CodeEditor from '@/components/Textarea/CodeEditor';
import { graphql } from '@/gql';
import { EnvironmentType } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';

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
  payload: { name: string; data: {} };
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
  const hasEventKey = Boolean(eventKey);
  return [
    {
      tabLabel: 'JSON Editor',
      tabTitle: 'Send Custom JSON',
      submitButtonLabel: 'Send Event',
      submitButtonEnabled: hasEventKey,
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
  const [payload, setPayload] = useState({ name: eventName, data: {} });
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
  const [tabs, setTabs] = useState(
    buildTabs({
      envName,
      payload,
      eventKey,
      sendEventURL,
      isBranchChild,
      copyToClipboardAction,
      sendEventAction,
    })
  );

  //
  // serialize data to state on change so we can persist it between editor tab changes
  const serializeData = (code: string) => {
    try {
      //
      // look for string like has json in it with name and data fields
      const matches = code.match(
        /\{[^{}]*"name"\s*:\s*"[^"]*"[^{}]*"data"\s*:\s*\{[^{}]*\}[^{}]*\}/g
      );
      const parsed = matches && JSON.parse(matches[0]);
      setPayload(parsed);
    } catch (error) {
      console.info("can't parse code editor payload, skipping serialization");
    }
  };

  async function sendEventAction(event: React.FormEvent<HTMLFormElement>) {
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
  }

  function copyToClipboardAction(event: React.FormEvent<HTMLFormElement>) {
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
  }

  return (
    <Modal className="max-w-6xl space-y-3 p-6" isOpen={isOpen} onClose={onClose}>
      <Tab.Group
        as="section"
        className="space-y-6"
        onChange={() => {
          //
          // rebuild tabs on change to carry over any payload
          setTabs(
            buildTabs({
              envName,
              payload,
              eventKey,
              sendEventURL,
              isBranchChild,
              copyToClipboardAction,
              sendEventAction,
            })
          );
        }}
      >
        <header className="flex items-center justify-between">
          <span className="inline-flex items-center gap-2">
            <h2 className="text-lg font-medium">Send Event</h2>
          </span>
          <Tab.List className="flex items-center gap-1 rounded-lg bg-slate-50 p-1">
            {tabs.map(({ tabLabel }) => (
              <Tab
                key={tabLabel}
                className="ui-selected:bg-white ui-selected:shadow-outline-secondary-light ui-selected:text-slate-700 ui-selected:hover:bg-white ui-selected:hover:text-slate-700 rounded px-3 py-1 text-sm font-medium text-slate-400 hover:bg-slate-100 hover:text-indigo-500"
              >
                {tabLabel}
              </Tab>
            ))}
          </Tab.List>
        </header>
        {!hasEventKey && (
          <Alert severity="warning">
            There are no Event Keys for this environment. Please create an Event Key in{' '}
            <Link href={pathCreator.keys({ envSlug: environment.slug })} className="underline">
              the Manage tab
            </Link>{' '}
            first.
          </Alert>
        )}
        <Tab.Panels>
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
              <Tab.Panel
                key={tabLabel}
                as="form"
                className="rounded-md bg-slate-900"
                onSubmit={submitAction}
              >
                <header className="bg bg-slate-910 flex items-center justify-between rounded-t-md p-2">
                  <h3 className="px-2 text-white">{tabTitle}</h3>
                  <NewButton
                    type="submit"
                    disabled={!submitButtonEnabled}
                    label={submitButtonLabel}
                    kind="primary"
                  />
                </header>
                <div className="w-full overflow-auto p-4">
                  <CodeEditor
                    language={codeLanguage}
                    initialCode={initialCode}
                    name="code"
                    className="h-80 w-[640px] bg-slate-900"
                    onCodeChange={serializeData}
                  />
                </div>
              </Tab.Panel>
            )
          )}
        </Tab.Panels>
      </Tab.Group>
      <NewButton onClick={onClose} appearance="outlined" label="Close Modal" />
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
