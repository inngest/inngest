'use client';

import NextLink from 'next/link';
import { Alert } from '@inngest/components/Alert';
import { Time } from '@inngest/components/Time';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { RiArrowRightSLine } from '@remixicon/react';

import Block from '@/components/Block';
import SimpleBarChart from '@/components/Charts/SimpleBarChart';
import LatestLogsList from '@/components/Events/LatestLogsList';
import Overlay from '@/components/Overlay';
import { useEventType } from '@/queries';

type EventDashboardProps = {
  params: {
    environmentSlug: string;
    eventName: string;
  };
};

export const runtime = 'nodejs';

export default function EventDashboard({ params }: EventDashboardProps) {
  const [{ data, fetching }] = useEventType({
    name: decodeURIComponent(params.eventName),
  });

  const eventNameDecoded = decodeURIComponent(params.eventName);
  const { eventType, dailyUsage } = data || {};

  return (
    <div className="grid-cols-dashboard bg-canvasSubtle grid min-h-0 flex-1">
      {!eventType && !fetching && (
        <Overlay>
          <div className="mt-20 flex place-content-center">
            <Alert severity="warning">
              This event has not been received by the {params.environmentSlug} environment.
            </Alert>
          </div>
        </Overlay>
      )}
      <main className="col-span-3 overflow-y-auto">
        <SimpleBarChart
          title={<>Events volume</>}
          period="24 Hours"
          data={dailyUsage}
          legend={[
            {
              name: 'Events',
              dataKey: 'count',
              color: 'rgb(var(--color-primary-subtle) / 1)',
              default: true,
            },
          ]}
          total={eventType?.usage.total || 0}
          totalDescription="24 Hour Volume"
          loading={fetching}
        />
        <LatestLogsList environmentSlug={params.environmentSlug} eventName={eventNameDecoded} />
      </main>
      <aside className="border-subtle bg-canvasSubtle overflow-y-auto border border-t-0 px-6 py-4">
        <Block title="Triggered Functions">
          {eventType && eventType.workflows.length > 0
            ? eventType.workflows.map((w) => (
                <NextLink
                  href={`/env/${params.environmentSlug}/functions/${encodeURIComponent(w.slug)}`}
                  key={w.id}
                  className="border-subtle bg-canvasBase hover:bg-canvasMuted mb-4 block overflow-hidden rounded border p-4"
                >
                  <div className="flex min-w-0 items-center">
                    <div className="min-w-0 flex-1">
                      <div className="flex min-w-0 items-center">
                        <FunctionsIcon className="h-5 w-5 pr-2" />
                        <p className="truncate font-medium">{w.name}</p>
                      </div>

                      {w.current?.createdAt && (
                        <Time
                          className="text-subtle text-xs"
                          format="relative"
                          value={new Date(w.current.createdAt)}
                        />
                      )}
                    </div>
                    <RiArrowRightSLine className="h-5" />
                  </div>
                </NextLink>
              ))
            : !fetching && (
                <p className="my-4 text-sm leading-6">No functions triggered by this event.</p>
              )}
        </Block>
      </aside>
    </div>
  );
}
