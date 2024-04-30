'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { ExclamationTriangleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { Secret } from '@/components/Secret';
import { graphql } from '@/gql';
import VercelLogomark from '@/logos/vercel-logomark.svg';
import { pathCreator } from '@/utils/urls';
import { InlineCode } from '../manage/signing-key/InlineCode';

const GetEventKeysForBlankSlateDocument = graphql(`
  query GetEventKeysForBlankSlate($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      ingestKeys(filter: { source: "key" }) {
        name
        presharedKey
        createdAt
      }
    }
  }
`);

function getDefaultEventKey<T extends { createdAt: string; name: null | string }>(
  keys: T[]
): T | undefined {
  const def = keys.find((k) => k.name && k.name.match(/default/i));
  return (
    def ||
    [...keys].sort((a, b) => {
      return Date.parse(a.createdAt) - Date.parse(b.createdAt);
    })[0]
  );
}

export default function EventListNotFound() {
  const router = useRouter();
  const environment = useEnvironment();
  const [{ data }] = useQuery({
    query: GetEventKeysForBlankSlateDocument,
    variables: {
      environmentID: environment.id,
    },
  });
  const ingestKey = getDefaultEventKey(data?.environment.ingestKeys || []);
  const key = ingestKey?.presharedKey;

  return (
    <div className="h-full w-full overflow-y-scroll py-16">
      <div className="mx-auto flex w-[640px] flex-col gap-4">
        <div className="text-center">
          <h3 className="mb-4 flex items-center justify-center gap-1 rounded-lg border border-indigo-100 bg-indigo-50 py-2.5 text-lg font-semibold text-indigo-500">
            <ExclamationTriangleIcon className="mt-0.5 h-5 w-5 text-indigo-700" />
            <span>
              No Events <span className="font-medium text-indigo-900">received in</span>{' '}
              {environment.name}
            </span>
          </h3>
        </div>

        <Card>
          <Card.Content>
            <div>
              <h3 className="mb-2 flex items-center text-xl font-medium">
                <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-700 text-center text-sm text-white">
                  1
                </span>
                Send your events
              </h3>
              <p className="mb-4 text-sm tracking-wide">
                After syncing your app, you can start sending events to this environment. To send
                events, your application needs to have an Event Key.
              </p>

              <h4 className="mb-2 text-base font-semibold">Event Key</h4>
              <p className="mb-2 text-sm tracking-wide">
                An Event Key let&apos;s your application send events to Inngest with{' '}
                <InlineCode value="inngest.send(...)" />.
              </p>
              <p className="mb-4 text-sm tracking-wide">
                We recommend adding your event key as the <InlineCode value="INNGEST_EVENT_KEY" />.
                environment variable in your application. Your platform may support setting this in
                an <InlineCode value=".env" />. file or you may need to set it manually on your
                platform.
              </p>

              {key && <Secret kind="event-key" secret={key} />}
            </div>

            <div className="mt-6 flex items-center gap-2">
              <Button
                kind="primary"
                href={pathCreator.apps({ envSlug: environment.slug })}
                label="Sync Your App"
              />
              <div className="flex gap-2 pl-2">
                <Button
                  appearance="outlined"
                  href={
                    'https://www.inngest.com/docs/deploy/vercel?ref=app-onboarding-events' as Route
                  }
                  target="_blank"
                  rel="noreferrer"
                  icon={<VercelLogomark className="-ml-0.5" />}
                  label="Vercel Integration"
                />
                <Button
                  appearance="outlined"
                  target="_blank"
                  href={'https://www.inngest.com/docs/events?ref=app-onboarding-events' as Route}
                  label="Read The Docs"
                />
              </div>
            </div>
          </Card.Content>
        </Card>

        <Card>
          <Card.Content>
            <h3 className="flex items-center text-xl font-semibold text-slate-800">
              <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-800 text-center text-sm text-white">
                2
              </span>
              Trigger Functions With Events
            </h3>
            <p className="mt-2 text-sm font-medium text-slate-500">
              After registering your functions, you can trigger them with events sent to this
              environment. Your events will show up on this page when they are received.
            </p>
            <div className="mt-6 flex items-center gap-2">
              <Button
                appearance="outlined"
                target="_blank"
                btnAction={() => router.refresh()}
                label="Refresh page to check for events"
              />
            </div>
          </Card.Content>
        </Card>
      </div>
    </div>
  );
}
