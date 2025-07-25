'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import NextLink from 'next/link';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { usePathCreator } from '@inngest/components/SharedContext/usePathCreator';
import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';
import { type Event } from '@inngest/components/types/event';
import { cn } from '@inngest/components/utils/classNames';
import { devServerURL, useDevServer } from '@inngest/components/utils/useDevServer';
import { RiArrowRightSLine, RiExternalLinkLine } from '@remixicon/react';
import { useQuery } from '@tanstack/react-query';

import { CodeBlock } from '../CodeBlock';
import {
  IDElement,
  LazyElementWrapper,
  PillElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/NewElement';
import { Link } from '../Link';
import { useShared } from '../SharedContext/SharedContext';
import { StatusDot } from '../Status/StatusDot';
import { DragDivider } from '../icons/DragDivider';
import { loadingSentinel, type Lazy } from '../utils/lazyLoad';
import type { EventsTable } from './EventsTable';

function toLazy<T>(data: T | undefined, isPending: boolean): Lazy<T | undefined> {
  return isPending ? loadingSentinel : data;
}

export function EventDetails({
  initialData,
  eventID,
  getEventDetails,
  getEventPayload,
  getEventRuns,
  expandedRowActions,
  standalone,
  pollInterval,
  autoRefresh,
}: {
  initialData?: Pick<Event, 'name' | 'runs'>;
  eventID: string;
  getEventDetails: React.ComponentProps<typeof EventsTable>['getEventDetails'];
  getEventPayload: React.ComponentProps<typeof EventsTable>['getEventPayload'];
  getEventRuns?: ({ eventID }: { eventID: string }) => Promise<Pick<Event, 'runs' | 'name'>>;
  expandedRowActions: React.ComponentProps<typeof EventsTable>['expandedRowActions'];
  standalone: boolean;
  pollInterval?: number;
  autoRefresh?: boolean;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const leftColumnRef = useRef<HTMLDivElement>(null);
  const eventInfoRef = useRef<HTMLDivElement>(null);
  const [leftWidth, setLeftWidth] = useState(70);
  const [isDragging, setIsDragging] = useState(false);
  const { isRunning, send } = useDevServer();
  const { pathCreator } = usePathCreator();
  const { cloud } = useShared();

  const {
    isPending, // first load, no data
    error,
    data: eventDetailsData,
    refetch: refetchEventDetails,
  } = useQuery({
    queryKey: ['event-details', { eventID: eventID }],
    queryFn: useCallback(() => {
      return getEventDetails({ eventID: eventID });
    }, [getEventDetails, eventID]),
    refetchInterval: autoRefresh ? pollInterval : false,
  });

  const {
    isPending: isPendingPayload,
    error: payloadError,
    data: eventPayloadData,
    refetch: refetchPayload,
  } = useQuery({
    queryKey: ['event-payload', { eventID: eventID }],
    queryFn: useCallback(() => {
      return getEventPayload({ eventID: eventID });
    }, [getEventPayload, eventID]),
    refetchInterval: autoRefresh ? pollInterval : false,
  });

  const {
    isPending: isPendingRuns,
    error: runsError,
    data: eventRunsData,
    refetch: refetchRuns,
  } = useQuery({
    queryKey: ['event-runs', { eventID: eventID }],
    queryFn: useCallback(() => {
      if (!getEventRuns) {
        return Promise.reject(new Error('getEventRuns is not defined'));
      }
      return getEventRuns({ eventID });
    }, [getEventRuns, eventID]),
    enabled: !!getEventRuns,
    refetchInterval: autoRefresh ? pollInterval : false,
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
    return <ErrorCard error={error} reset={() => refetchEventDetails()} />;
  }

  const prettyPayload =
    usePrettyJson(eventPayloadData?.payload ?? '') || (eventPayloadData?.payload ?? '');

  const eventName = initialData?.name || eventDetailsData?.name;
  const eventRuns = initialData?.runs || eventRunsData?.runs;

  return (
    <>
      {standalone && (
        <div className="flex flex-row items-start justify-between px-4 pb-4 pt-8">
          <div className="flex flex-col gap-1">
            {(isPending || isPendingRuns) && !eventName ? (
              <Skeleton className="block h-8 w-64" />
            ) : (
              <p className="text-basis text-2xl font-medium">{eventName}</p>
            )}
            <p className="text-subtle font-mono">{eventID}</p>
          </div>
        </div>
      )}

      <div
        ref={containerRef}
        className={cn('flex flex-row', standalone ? 'border-subtle border-t' : '')}
      >
        <div ref={leftColumnRef} className="flex flex-col gap-2" style={{ width: `${leftWidth}%` }}>
          <div ref={eventInfoRef} className="flex flex-col">
            <div className="mb-3 flex h-8 items-center justify-between gap-1 px-4">
              <div className="flex items-center gap-2">
                <p className="text-basis text-base">{eventName}</p>
                {!standalone && (
                  <Link
                    size="medium"
                    target="_blank"
                    href={pathCreator.eventPopout({ eventID: eventID })}
                    iconAfter={<RiExternalLinkLine className="h-4 w-4 shrink-0" />}
                  />
                )}
              </div>
              {expandedRowActions({
                eventName: eventName,
                payload: eventPayloadData?.payload,
              })}
            </div>
            <div className="mb-3 flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4 px-4">
              <LazyElementWrapper
                label="Event ID"
                lazy={toLazy(eventDetailsData?.id, isPending)}
                className="min-w-56"
              >
                {(id) => <IDElement>{id || '-'}</IDElement>}
              </LazyElementWrapper>
              <LazyElementWrapper
                label="Idempotency key"
                lazy={toLazy(eventDetailsData?.idempotencyKey, isPending)}
              >
                {(idempotencyKey) => <TextElement>{idempotencyKey || '-'}</TextElement>}
              </LazyElementWrapper>
              <LazyElementWrapper
                label="Source"
                lazy={toLazy(eventDetailsData?.source?.name, isPending)}
              >
                {(name) => <PillElement>{name || 'N/A'}</PillElement>}
              </LazyElementWrapper>
              <LazyElementWrapper label="TS" lazy={toLazy(eventDetailsData?.occurredAt, isPending)}>
                {(occurredAt) =>
                  occurredAt ? (
                    <TimeElement date={new Date(occurredAt)} />
                  ) : (
                    <TextElement>-</TextElement>
                  )
                }
              </LazyElementWrapper>
              <LazyElementWrapper
                label="Version"
                lazy={toLazy(eventDetailsData?.version, isPending)}
              >
                {(version) => <TextElement>{version || '-'}</TextElement>}
              </LazyElementWrapper>
            </div>
            {!payloadError && (
              <div className="border-subtle border-t pl-px">
                <CodeBlock
                  loading={isPendingPayload}
                  header={{ title: 'Payload' }}
                  tab={{
                    content: prettyPayload,
                  }}
                  allowFullScreen={true}
                  actions={
                    cloud
                      ? [
                          {
                            label: 'Send to Dev Server',
                            title: isRunning
                              ? 'Send event payload to running Dev Server'
                              : `Dev Server is not running at ${devServerURL}`,
                            onClick: () => send(eventPayloadData?.payload || ''),
                            disabled: !isRunning,
                          },
                        ]
                      : []
                  }
                />
              </div>
            )}
            {payloadError && <ErrorCard error={payloadError} reset={() => refetchPayload()} />}
          </div>
        </div>

        <div className="relative cursor-col-resize" onMouseDown={handleMouseDown}>
          <div className="bg-canvasMuted absolute inset-0 z-[1] h-full w-px" />
          <div
            className="absolute z-[1] -translate-x-1/2"
            style={{
              top: (eventInfoRef.current?.clientHeight ?? 0) / 2,
            }}
          >
            <DragDivider className="bg-canvasBase" />
          </div>
        </div>

        <div
          className="border-muted flex flex-col justify-start"
          style={{ width: `${100 - leftWidth}%` }}
        >
          <div className="px-4 py-2">
            <p className="text-muted mb-4 text-xs font-medium uppercase">Functions Triggered</p>
            {runsError ? (
              <ErrorCard error={runsError} reset={() => refetchRuns()} />
            ) : isPendingRuns && !eventRuns ? (
              <Skeleton className="block h-12 w-full p-1.5" />
            ) : eventRuns?.length ? (
              <ul className="divide-light divide-y [&>*:not(:first-child)]:pt-[6px] [&>*:not(:last-child)]:pb-[6px]">
                {eventRuns.map((run) => (
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
    </>
  );
}
