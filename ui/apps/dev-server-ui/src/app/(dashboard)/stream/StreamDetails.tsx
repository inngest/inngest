import { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ContentCard } from '@inngest/components/ContentCard';
import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';
import { classNames } from '@inngest/components/utils/classNames';
import { ulid } from 'ulid';

import SendEventButton from '@/components/Event/SendEventButton';
import { useSendEventMutation } from '@/store/devApi';
import { useEvent } from './useEvent';
import { useGetHistoryItemOutput } from './useGetHistoryItemOutput';
import { useRun } from './useRun';

export default function StreamDetails() {
  const params = useSearchParams();
  const eventID = params.get('event');
  const runID = params.get('run');

  const eventResult = useEvent(eventID);
  if (eventResult.error) {
    throw eventResult.error;
  }

  const runResult = useRun(runID);
  if (runResult.error) {
    throw runResult.error;
  }

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
    }).unwrap();
  }

  return (
    <div
      className={classNames(
        'grid h-full text-white',
        eventResult.data ? 'grid-cols-2' : 'grid-cols-1'
      )}
    >
      {eventResult.data && (
        <EventDetails
          event={eventResult.data}
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
          func={runResult.data.func}
          getHistoryItemOutput={getHistoryItemOutput}
          history={runResult.data.history}
          run={runResult.data.run}
        />
      )}
    </div>
  );
}
