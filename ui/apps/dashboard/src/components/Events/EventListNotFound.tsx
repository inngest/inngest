'use client';

import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card';
import { InlineCode } from '@inngest/components/Code';
import { RiErrorWarningLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { Secret } from '@/components/Secret';
import VercelLogomark from '@/logos/vercel-logomark.svg';
import { useDefaultEventKey } from '@/queries/useDefaultEventKey';
import { pathCreator } from '@/utils/urls';

export default function EventListNotFound() {
  const router = useRouter();
  const environment = useEnvironment();
  const res = useDefaultEventKey({ envID: environment.id });
  const key = res.data?.defaultKey.presharedKey;

  return (
    <div className="text-basis h-full w-full overflow-y-scroll py-16">
      <div className="mx-auto flex w-[640px] flex-col gap-4">
        <div className="text-center">
          <h3 className="border-info bg-info text-info mb-4 flex items-center justify-center gap-1 rounded-md border py-2.5 text-lg font-semibold">
            <RiErrorWarningLine className="text-info h-5 w-5" />
            <span>
              No Events <span className="text-info font-normal">received in</span>{' '}
              {environment.name}
            </span>
          </h3>
        </div>

        <Card>
          <Card.Content>
            <div>
              <h3 className="mb-2 flex items-center text-xl font-medium">
                <span className="bg-canvasMuted mr-2 inline-flex h-6  w-6 items-center justify-center rounded-full text-center text-sm">
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
                <InlineCode>inngest.send(...)</InlineCode>.
              </p>
              <p className="mb-4 text-sm tracking-wide">
                We recommend adding your event key as the <InlineCode>INNGEST_EVENT_KEY</InlineCode>
                . environment variable in your application. Your platform may support setting this
                in an <InlineCode>.env</InlineCode>. file or you may need to set it manually on your
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
            <h3 className="flex items-center text-xl font-medium">
              <span className="bg-canvasMuted mr-2 inline-flex h-6  w-6 items-center justify-center rounded-full text-center text-sm">
                2
              </span>
              Trigger Functions With Events
            </h3>
            <p className="mt-2 text-sm">
              After registering your functions, you can trigger them with events sent to this
              environment. Your events will show up on this page when they are received.
            </p>
            <div className="mt-6 flex items-center gap-2">
              <Button
                appearance="outlined"
                kind="secondary"
                target="_blank"
                onClick={() => router.refresh()}
                label="Refresh page to check for events"
              />
            </div>
          </Card.Content>
        </Card>
      </div>
    </div>
  );
}
