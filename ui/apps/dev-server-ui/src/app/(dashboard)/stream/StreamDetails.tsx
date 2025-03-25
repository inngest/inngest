import React, { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ContentCard } from '@inngest/components/ContentCard';
import { EventDetails } from '@inngest/components/EventDetails';
import { Link } from '@inngest/components/Link';
import { RunDetails } from '@inngest/components/RunDetails';
import type { NavigateToRunFn } from '@inngest/components/Timeline/Timeline';
import { cn } from '@inngest/components/utils/classNames';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import SendEventButton from '@/components/Event/SendEventButton';
import { useSendEventMutation } from '@/store/devApi';
import { useCancelRunMutation, useRerunMutation } from '@/store/generated';
import { useEvent } from './useEvent';
import { useGetHistoryItemOutput } from './useGetHistoryItemOutput';
import { useRun } from './useRun';

export default function StreamDetails() {
  const params = useSearchParams();
  const eventID = params.get('event');
  const runID = params.get('run');
  const [cancelRun] = useCancelRunMutation();
  const [rerun] = useRerunMutation();

  const eventResult = useEvent(eventID);
  useEffect(() => {
    if (eventResult.error) {
      console.error(eventResult.error);
      toast.error(`Failed to fetch event ${eventID}`);
    }
  }, [eventResult.error]);

  const runResult = useRun(runID);
  useEffect(() => {
    if (runResult.error) {
      console.error(runResult.error);
      toast.error(`Failed to fetch run ${runID}`);
    }
  }, [runResult.error]);

  useEffect(() => {
    if (eventResult.error || runResult.error) {
      // If there's an error fetching the event or run, we should redirect to the
      // stream. This happens a lot, since restarting the Dev Server will clear
      // all data
      router.replace('/stream');
    }
  }, [eventResult.error, runResult.error]);

  const getHistoryItemOutput = useGetHistoryItemOutput(runID);
  const router = useRouter();

  const [selectedRunID, setSelectedRunID] = useState<string | undefined>(runID ?? undefined);
  const [sendEvent] = useSendEventMutation();

  useEffect(() => {
    if (!selectedRunID && eventResult.data?.functionRuns[0]) {
      const firstRunID = eventResult.data.functionRuns[0].id;
      setSelectedRunID(firstRunID);
    }
  }, [selectedRunID, eventResult.data?.functionRuns]);

  const renderSendEventButton = useMemo(() => {
    return () => (
      <SendEventButton
        label="Edit and Replay"
        appearance="outlined"
        data={eventResult.data?.payload}
      />
    );
  }, [eventResult.data?.payload]);

  if (eventResult.isLoading || runResult.isLoading) {
    return (
      <ContentCard>
        <div className="flex h-full w-full items-center justify-center p-8">
          <div className="italic opacity-75">Loading...</div>
        </div>
      </ContentCard>
    );
  }

  function onReplayEvent() {
    if (!eventResult.data?.payload) {
      return;
    }

    const eventId = ulid();

    sendEvent({
      ...JSON.parse(eventResult.data.payload),
      id: eventId,
      ts: Date.now(),
    })
      .unwrap()
      .then(() => {
        toast.success('The event was successfully replayed.');
      })
      .catch(() => {
        toast.error('Failed to replay');
      });
  }

  const navigateToRun: NavigateToRunFn = (opts) => {
    const runParams = new URLSearchParams({
      event: opts.eventID,
      run: opts.runID,
    });

    return (
      <Link size="small" arrowOnHover href={`/stream/trigger?${runParams.toString()}`}>
        Go to run
      </Link>
    );
  };

  let events: React.ComponentProps<typeof EventDetails>['events'] = [];
  if (runResult?.data?.run?.events && runResult.data.run.events.length > 1) {
    events = runResult.data.run.events;
  } else if (eventResult.data) {
    events = [eventResult.data];
  }

  return (
    <div className={cn('text-basis grid h-full', eventResult.data ? 'grid-cols-2' : 'grid-cols-1')}>
      {eventResult.data && (
        <EventDetails
          batchCreatedAt={runResult.data?.run?.batchCreatedAt ?? undefined}
          batchID={runResult.data?.run?.batchID ?? undefined}
          events={events}
          functionRuns={eventResult.data.functionRuns}
          onFunctionRunClick={(runId) => {
            setSelectedRunID(runId);
            router.push(`/stream/trigger?event=${eventResult.data.id}&run=${runId}`);
          }}
          onReplayEvent={onReplayEvent}
          selectedRunID={selectedRunID}
          SendEventButton={renderSendEventButton}
        />
      )}

      {runResult.data && (
        <RunDetails
          cancelRun={async () => {
            const res = await cancelRun({ runID: runResult.data.run.id });
            if ('error' in res) {
              // Throw error so that the modal can catch and display it
              throw res.error;
            }
          }}
          func={runResult.data.func}
          getHistoryItemOutput={getHistoryItemOutput}
          history={runResult.data.history}
          rerun={async () => {
            const res = await rerun({ runID: runResult.data.run.id });
            if ('error' in res) {
              // Throw error so that the modal can catch and display it
              throw res.error;
            }
          }}
          run={runResult.data.run}
          navigateToRun={navigateToRun}
        />
      )}
    </div>
  );
}
