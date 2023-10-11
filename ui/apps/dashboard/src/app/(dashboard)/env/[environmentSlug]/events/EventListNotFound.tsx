'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { ExclamationTriangleIcon } from '@heroicons/react/20/solid';
import { useCopyToClipboard } from 'react-use';
import { useQuery } from 'urql';

import Button from '@/components/Button';
import SecretKey from '@/components/Secrets/SecretKey';
import { graphql } from '@/gql';
import VercelLogomark from '@/logos/vercel-logomark.svg';
import { useEnvironment } from '@/queries';

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

export default function EventListNotFound({ environmentSlug }: { environmentSlug: string }) {
  const router = useRouter();
  const [, copy] = useCopyToClipboard();
  const [{ data: environment, fetching: fetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [{ data, fetching: fetchingKey }] = useQuery({
    query: GetEventKeysForBlankSlateDocument,
    variables: {
      environmentID: environment?.id!,
    },
    pause: !environment?.id,
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
              {environment?.name}
            </span>
          </h3>
        </div>
        <div className="bg-slate-950 rounded-lg px-8 pt-8">
          <div className="bg-slate-950/20 -mt-8 pt-6 backdrop-blur-sm">
            <h3 className="flex items-center text-xl font-medium text-white">
              <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-700 text-center text-sm text-white">
                1
              </span>
              Send your events
            </h3>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              After deploying your functions, you can start sending events to this environment. To
              send events, your application needs to have an Event Key.
            </p>
            <h4 className="mt-4 text-base font-semibold text-white">Event Key</h4>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              An Event Key let&apos;s your application send events to Inngest with{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                inngest.send(...)
              </code>
              .
            </p>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              We recommend adding your event key as the{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                INNGEST_EVENT_KEY
              </code>{' '}
              environment variable in your application. Your platform may support setting this in an{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                .env
              </code>{' '}
              file or you may need to set it manually on your platform.
            </p>
            <SecretKey
              value={key || '...'}
              masked={`${key?.substring(0, 6)}-<click-to-reveal>`}
              className="mt-4"
              context="dark"
            />
          </div>

          <div className="mt-6 flex items-center gap-2 border-t border-slate-800/50 py-4">
            <Button variant="primary" href={`/env/${environmentSlug}/deploys` as Route}>
              Deploy Your Functions
            </Button>
            <div className="flex gap-2 border-l border-slate-800/50 pl-2">
              <Button
                href={
                  'https://www.inngest.com/docs/deploy/vercel?ref=app-onboarding-events' as Route
                }
                target="_blank"
                rel="noreferrer"
                variant="secondary"
                context="dark"
              >
                <VercelLogomark className="-ml-0.5 h-4 w-4" />
                Vercel Integration
              </Button>
              <Button
                variant="secondary"
                context="dark"
                target="_blank"
                href={'https://www.inngest.com/docs/events?ref=app-onboarding-events' as Route}
              >
                Read The Docs
              </Button>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-slate-300 px-8 pt-8">
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
          <div className="mt-6 flex items-center gap-2 border-t border-slate-100 py-4">
            <Button variant="secondary" target="_blank" onClick={() => router.refresh()}>
              Refresh page to check for events
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
