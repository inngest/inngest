'use client';

import { use } from 'react';
import NextLink from 'next/link';
import { useEventTypeVolume } from '@inngest/components/EventTypes/useEventTypeVolume';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { RiArrowRightSLine } from '@remixicon/react';

import Block from '@/components/Block';
import SimpleBarChart from '@/components/Charts/SimpleBarChart';
import {
  useEventTypeVolume as getEventTypeVolume,
  useEventType,
} from '@/components/EventTypes/useEventTypes';
import LatestLogsList from '@/components/Events/LatestLogsList';

type EventDashboardProps = {
  params: Promise<{
    environmentSlug: string;
    eventTypeName: string;
  }>;
};

export const runtime = 'nodejs';

export default function EventDashboard(props: EventDashboardProps) {
  const params = use(props.params);
  const eventNameDecoded = decodeURIComponent(params.eventTypeName);
  const { data, isLoading: isLoadingVolume } = useEventTypeVolume(
    eventNameDecoded,
    getEventTypeVolume()
  );
  const { data: eventType, isLoading } = useEventType({ eventName: eventNameDecoded });

  const { volume } = data || {};

  const parsedVolumeData = volume?.dailyVolumeSlots.map((slot) => ({
    name: slot.slot,
    values: {
      count: slot.startCount,
    },
  }));

  return (
    <div className="grid-cols-dashboard bg-canvasSubtle grid min-h-0 flex-1">
      <main className="col-span-3 overflow-y-auto">
        <SimpleBarChart
          title={<>Events volume</>}
          period="24 Hours"
          data={parsedVolumeData}
          legend={[
            {
              name: 'Events',
              dataKey: 'count',
              color: 'rgb(var(--color-primary-subtle) / 1)',
              default: true,
            },
          ]}
          total={volume?.totalVolume || 0}
          totalDescription="24 Hour Volume"
          loading={isLoadingVolume}
        />
        <LatestLogsList environmentSlug={params.environmentSlug} eventName={eventNameDecoded} />
      </main>
      <aside className="border-subtle bg-canvasSubtle overflow-y-auto border border-t-0 px-6 py-4">
        <Block title="Triggered Functions">
          {eventType && eventType.functions.length > 0
            ? eventType.functions.map((f) => (
                <NextLink
                  href={`/env/${params.environmentSlug}/functions/${encodeURIComponent(f.slug)}`}
                  key={f.id}
                  className="border-subtle bg-canvasBase hover:bg-canvasMuted mb-4 block overflow-hidden rounded border p-4"
                >
                  <div className="flex min-w-0 items-center">
                    <div className="min-w-0 flex-1">
                      <div className="flex min-w-0 items-center">
                        <FunctionsIcon className="h-5 w-5 pr-2" />
                        <p className="truncate font-medium">{f.name}</p>
                      </div>
                    </div>
                    <RiArrowRightSLine className="h-5" />
                  </div>
                </NextLink>
              ))
            : !isLoading && (
                <p className="my-4 text-sm leading-6">No functions triggered by this event.</p>
              )}
        </Block>
      </aside>
    </div>
  );
}
