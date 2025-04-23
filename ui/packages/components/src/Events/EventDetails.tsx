import { useCallback, useEffect, useRef, useState } from 'react';
import NextLink from 'next/link';
import { Time } from '@inngest/components/Time';
import { type Event } from '@inngest/components/types/event';
import { RiArrowRightSLine } from '@remixicon/react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { type Row } from '@tanstack/react-table';

import {
  ElementWrapper,
  IDElement,
  PillElement,
  SkeletonElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/NewElement';
import { IO } from '../RunDetailsV3/IO';
import { Tabs } from '../RunDetailsV3/Tabs';
import { StatusDot } from '../Status/StatusDot';
import { DragDivider } from '../icons/DragDivider';
import type { EventsTable } from './EventsTable';

export function EventDetails({
  row,
  getEventDetails,
  pathCreator,
  expandedRowActions,
}: {
  row: Row<Omit<Event, 'payload'>>;
  pathCreator: React.ComponentProps<typeof EventsTable>['pathCreator'];
  getEventDetails: React.ComponentProps<typeof EventsTable>['getEventDetails'];
  expandedRowActions: React.ComponentProps<typeof EventsTable>['expandedRowActions'];
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const eventInfoRef = useRef<HTMLDivElement>(null);
  const [leftWidth, setLeftWidth] = useState(55);
  const [isDragging, setIsDragging] = useState(false);
  const [height, setHeight] = useState(0);
  const MIN_HEIGHT = 186;

  const {
    isPending, // first load, no data
    error,
    data: eventDetailsData,
  } = useQuery({
    queryKey: ['event-details', { eventName: row.original.name }],
    queryFn: useCallback(() => {
      return getEventDetails({ eventName: row.original.name });
    }, [getEventDetails, row.original.name]),
    placeholderData: keepPreviousData,
    refetchInterval: 5000,
  });

  const handleMouseDown = useCallback(() => {
    setIsDragging(true);
  }, []);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging) {
        return;
      }

      const container = containerRef.current;
      if (!container) {
        return;
      }

      const containerRect = container.getBoundingClientRect();
      const newWidth = ((e.clientX - containerRect.left) / containerRect.width) * 100;
      setLeftWidth(Math.min(Math.max(newWidth, 20), 80));
    },
    [isDragging]
  );

  useEffect(() => {
    //
    // left column height is dynamic and should determine right column height
    const h = leftColumnRef.current?.clientHeight ?? 0;
    setHeight(h > MIN_HEIGHT ? h : MIN_HEIGHT);
  }, [leftColumnRef.current?.clientHeight]);

  useEffect(() => {
    if (isDragging) {
      document.body.style.userSelect = 'none';
      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);
    }
    return () => {
      document.body.style.userSelect = '';
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  if (error) {
    // TODO: error handling
    console.log(error.message);
  }

  return (
    <div ref={containerRef} className="flex flex-row">
      <div ref={leftColumnRef} className="flex flex-col gap-2" style={{ width: `${leftWidth}%` }}>
        <div ref={eventInfoRef} className="flex flex-col gap-3">
          <div className="flex h-8 items-center justify-between gap-1 px-4">
            <p className="text-sm">{row.original.name}</p>
            {expandedRowActions(row.original.name)}
          </div>
          <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
            <ElementWrapper label="Event ID">
              {isPending ? <SkeletonElement /> : <IDElement>{eventDetailsData?.id}</IDElement>}
            </ElementWrapper>
            <ElementWrapper label="Idempotency Key">
              {isPending ? (
                <SkeletonElement />
              ) : (
                <TextElement>{eventDetailsData?.idempotencyKey || '-'}</TextElement>
              )}
            </ElementWrapper>
            <ElementWrapper label="Source">
              {isPending ? (
                <SkeletonElement />
              ) : (
                <PillElement>{eventDetailsData?.source || 'N/A'}</PillElement>
              )}
            </ElementWrapper>
            <ElementWrapper label="TS">
              {isPending ? (
                <SkeletonElement />
              ) : eventDetailsData?.timestamp ? (
                <TimeElement date={new Date(eventDetailsData.timestamp)} />
              ) : (
                <TextElement>-</TextElement>
              )}
            </ElementWrapper>
            <ElementWrapper label="Version">
              {isPending ? (
                <SkeletonElement />
              ) : (
                <TextElement>{eventDetailsData?.version || '-'}</TextElement>
              )}
            </ElementWrapper>
          </div>
          <Tabs
            defaultActive="rawPayload"
            tabs={[
              {
                label: 'Raw payload',
                id: 'rawPayload',
                node: (
                  <IO
                    title="Payload"
                    raw={
                      '{\n  "name": "signup.new",\n  "data": {\n    "account_id": "119f5971-9878-46bd-a18f-4fecd",\n    "method": "",\n    "plan_name": "Free Tier"\n  },\n  "id": "119f5971-9878-46bd-a18f-4f0680174ecd",\n  "ts": 1711051784369,\n  "v": "2021-05-11.01"\n}'
                    }
                  ></IO>
                ),
              },
              {
                label: 'Formatted data',
                id: 'formattedData',
                node: <p></p>,
              },
            ]}
          />
        </div>
      </div>

      <div className="relative cursor-col-resize" onMouseDown={handleMouseDown}>
        <div className="bg-canvasMuted absolute inset-0 z-[1] h-full w-px" />
        <div
          className="absolute z-[1] -translate-x-1/2"
          style={{
            top:
              (eventInfoRef.current?.clientHeight ?? 0) +
              (height - (eventInfoRef.current?.clientHeight ?? 0)) / 2,
          }}
        >
          <DragDivider className="bg-canvasBase" />
        </div>
      </div>

      <div
        className="border-muted flex flex-col justify-start"
        style={{ width: `${100 - leftWidth}%`, height: height }}
      >
        <div className="px-4 py-2">
          <p className="text-muted mb-4 text-xs font-medium uppercase">Functions Triggered</p>
          {eventDetailsData?.runs?.length ? (
            <ul className="divide-subtle divide-y [&>*:not(:first-child)]:pt-[6px] [&>*:not(:last-child)]:pb-[6px]">
              {/* TODO:  optimistically render the name and status from the row prop */}
              {eventDetailsData.runs.map((run) => (
                <li key={run.fnSlug}>
                  <NextLink
                    href={pathCreator.runPopout({ runID: run.id })}
                    className="hover:bg-canvasSubtle flex items-center justify-between rounded p-1.5"
                  >
                    <div className="flex flex-col gap-0.5">
                      <div className="flex items-center gap-2">
                        <StatusDot status={run.status} />
                        <p className="text-basis text-sm font-medium">{run.fnName}</p>
                      </div>
                      <div className="ml-[1.375rem] flex items-center gap-1">
                        <p className="text-subtle text-xs lowercase first-letter:capitalize">
                          {run.status}
                        </p>
                        {(run.completedAt || run.startedAt) && (
                          <Time
                            className="text-subtle text-xs"
                            format="relative"
                            value={run.completedAt ?? run.startedAt!}
                          />
                        )}
                      </div>
                    </div>
                    <RiArrowRightSLine className="text-muted h-5 shrink-0" />
                  </NextLink>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-subtle text-sm">No functions triggered by this event.</p>
          )}
        </div>
      </div>
    </div>
  );
}
