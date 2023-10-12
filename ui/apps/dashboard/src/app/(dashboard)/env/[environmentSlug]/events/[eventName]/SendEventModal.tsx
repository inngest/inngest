import type { Route } from 'next';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Tab } from '@headlessui/react';
import { PaperAirplaneIcon } from '@heroicons/react/20/solid';
import ky from 'ky';
import { toast } from 'sonner';
import { type JsonValue } from 'type-fest';
import { useQuery } from 'urql';

import { Alert } from '@/components/Alert';
import Button from '@/components/Button';
import Modal from '@/components/Modal';
import CodeEditor from '@/components/Textarea/CodeEditor';
import { graphql } from '@/gql';
import { EnvironmentType } from '@/gql/graphql';
import { useEnvironment } from '@/queries';

type UseDefaultEventKeyParams = {
  environmentSlug: string;
};

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
  environmentSlug: string;
  eventName?: string;
  isOpen: boolean;
  onClose: () => void;
};

export function SendEventModal({
  environmentSlug,
  eventName = 'Your Event Name',
  isOpen,
  onClose,
}: SendEventModalProps) {
  const router = useRouter();
  const [{ data: environment }] = useEnvironment({ environmentSlug });
  const eventKey = usePreferDefaultEventKey({ environmentSlug });
  const hasEventKey = Boolean(eventKey);
  const protocol = process.env.NODE_ENV === 'development' ? 'http://' : 'https://';
  const sendEventURL = `${protocol}${process.env.NEXT_PUBLIC_EVENT_API_HOST}/e/${
    eventKey || '<EVENT_KEY>'
  }`;

  const isBranchChild = environment?.type === EnvironmentType.BranchChild;
  const envName = environment?.name;

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

  const tabs = [
    {
      tabLabel: 'JSON Editor',
      tabTitle: 'Send Custom JSON',
      submitButtonLabel: 'Send Event',
      submitButtonEnabled: hasEventKey,
      submitAction: sendEventAction,
      codeLanguage: 'json',
      initialCode: JSON.stringify(
        {
          name: eventName,
          data: {},
        },
        null,
        2
      ),
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

await inngest.send({
  name: '${eventName}' // e.g. 'app/user.signed.up',
  data: {
    // Your event data
  }
});`,
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
  --data '{ "name": "${eventName}", "data": {} }'`,
    },
  ];

  return (
    <Modal className="max-w-6xl space-y-3 p-6" isOpen={isOpen} onClose={onClose}>
      <Tab.Group as="section" className="space-y-6">
        <header className="flex items-center justify-between">
          <span className="inline-flex items-center gap-2">
            <PaperAirplaneIcon className="h-5 text-indigo-500" />
            <h2 className="text-lg font-medium">Send Event</h2>
          </span>
          <Tab.List className="flex items-center gap-1 rounded-lg bg-slate-50 p-1">
            {tabs.map(({ tabLabel }) => (
              <Tab
                key={tabLabel}
                className="ui-selected:bg-white ui-selected:shadow-outline-secondary-light ui-selected:text-slate-700 ui-selected:hover:bg-white ui-selected:hover:text-slate-700 rounded-sm px-3 py-1 text-sm font-medium text-slate-400 hover:bg-slate-100 hover:text-indigo-500"
              >
                {tabLabel}
              </Tab>
            ))}
          </Tab.List>
        </header>
        {!hasEventKey && (
          <Alert severity="warning">
            There are no Event Keys for this environment. Please create an Event Key in{' '}
            <Link href={`/env/${environmentSlug}/manage/keys` as Route} className="underline">
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
                  <Button type="submit" context="dark" disabled={!submitButtonEnabled}>
                    {submitButtonLabel}
                  </Button>
                </header>
                <div className="w-full overflow-auto p-4">
                  <CodeEditor
                    language={codeLanguage}
                    initialCode={initialCode}
                    name="code"
                    className="h-80 w-[640px] bg-slate-900"
                  />
                </div>
              </Tab.Panel>
            )
          )}
        </Tab.Panels>
      </Tab.Group>
      <Button onClick={onClose} variant="secondary">
        Close Modal
      </Button>
    </Modal>
  );
}

function usePreferDefaultEventKey({
  environmentSlug,
}: UseDefaultEventKeyParams): string | undefined {
  const [{ data: environment }] = useEnvironment({
    environmentSlug,
  });
  const [{ data }] = useQuery({
    query: GetEventKeysDocument,
    variables: {
      environmentID: environment?.id!,
    },
    pause: !environment?.id,
  });

  const eventKeys = data?.environment.eventKeys;

  const defaultKey = eventKeys?.find((eventKey) => {
    return eventKey.name?.toLowerCase().startsWith('default in');
  })?.value;

  return defaultKey ?? eventKeys?.[0]?.value;
}
