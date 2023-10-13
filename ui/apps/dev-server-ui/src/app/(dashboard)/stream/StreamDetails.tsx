import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { EventSection } from '@inngest/components/RunDetails/EventSection';
import { ulid } from 'ulid';

import SendEventButton from '@/components/Event/SendEventButton';
import { FunctionRunSection } from '@/components/Function/RunSection';
import { useSendEventMutation } from '@/store/devApi';
import { useEvent } from './useEvent';

export default function StreamDetails() {
  const params = useSearchParams();
  const eventID = params.get('event');
  const cronID = params.get('cron');
  const runID = params.get('run');

  if (!eventID) {
    throw new Error('missing eventID');
  }

  const eventResult = useEvent(eventID);
  if (eventResult.error) {
    throw eventResult.error;
  }

  const [selectedRunID, setSelectedRunID] = useState<string | undefined>(runID ?? undefined);
  const [sendEvent] = useSendEventMutation();

  useEffect(() => {
    if (!selectedRunID && eventResult.data?.functionRuns[0]) {
      const firstRunID = eventResult.data.functionRuns[0].id;
      setSelectedRunID(firstRunID);
    }
  }, [selectedRunID, eventResult.data?.functionRuns]);

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

  const renderSendEventButton = useMemo(() => {
    return () => (
      <SendEventButton
        label="Edit and Replay"
        appearance="outlined"
        data={eventResult.data?.payload}
      />
    );
  }, [eventResult.data?.payload]);

  return (
    <>
      <div className="grid h-full grid-cols-2 text-white">
        {!eventResult.isLoading && (
          <EventSection
            event={eventResult.data}
            functionRuns={eventResult.data.functionRuns}
            onFunctionRunClick={setSelectedRunID}
            onReplayEvent={onReplayEvent}
            selectedRunID={selectedRunID}
            SendEventButton={renderSendEventButton}
          />
        )}

        <FunctionRunSection runId={runID} />
      </div>

      {cronID && runID && (
        <div className="grid h-full grid-cols-1 text-white">
          <FunctionRunSection runId={runID} />
        </div>
      )}
    </>
  );
}
