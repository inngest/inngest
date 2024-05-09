'use client';

import Link from 'next/link';
import { Alert } from '@inngest/components/Alert';
import { Time } from '@inngest/components/Time';
import { IconFunction } from '@inngest/components/icons/Function';
import { RiArrowRightSLine, RiBarChart2Fill } from '@remixicon/react';
import colors from 'tailwindcss/colors';

import Block from '@/components/Block';
import SimpleBarChart from '@/components/Charts/SimpleBarChart';
import Overlay from '@/components/Overlay';
import { useEventType } from '@/queries';
import LatestLogsList from './LatestLogsList';

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
    <div className="grid-cols-dashboard grid min-h-0 flex-1 bg-slate-100">
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
          title={
            <>
              <RiBarChart2Fill className="h-4 text-slate-700" /> Volume
            </>
          }
          period="24 Hours"
          data={dailyUsage}
          legend={[
            {
              name: 'Events',
              dataKey: 'count',
              color: colors.slate['600'],
              default: true,
            },
          ]}
          total={eventType?.usage.total || 0}
          totalDescription="24 Hour Volume"
          loading={fetching}
        />
        <LatestLogsList environmentSlug={params.environmentSlug} eventName={eventNameDecoded} />
      </main>
      <aside className="overflow-y-auto border border-slate-200 bg-white px-6 py-4">
        <Block title="Triggered Functions">
          {eventType && eventType.workflows.length > 0
            ? eventType.workflows.map((w) => (
                <Link
                  href={`/env/${params.environmentSlug}/functions/${encodeURIComponent(w.slug)}`}
                  key={w.id}
                  className="shadow-outline-secondary-light mb-4 block overflow-hidden rounded bg-white p-4 hover:bg-slate-50"
                >
                  <div className="flex min-w-0 items-center">
                    <div className="min-w-0 flex-1">
                      <div className="flex min-w-0 items-center">
                        <IconFunction className="h-5 pr-2 text-indigo-500" />
                        <p className="truncate font-medium">{w.name}</p>
                      </div>

                      {w.current?.createdAt && (
                        <Time
                          className="text-xs text-slate-500"
                          format="relative"
                          value={new Date(w.current.createdAt)}
                        />
                      )}
                    </div>
                    <RiArrowRightSLine className="h-5" />
                  </div>
                </Link>
              ))
            : !fetching && (
                <p className="my-4 text-sm leading-6 text-slate-700">
                  No functions triggered by this event.
                </p>
              )}
        </Block>
      </aside>
    </div>
  );
}
