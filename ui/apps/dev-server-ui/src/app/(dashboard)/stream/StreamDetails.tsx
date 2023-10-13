import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { EventSection } from '@inngest/components/RunDetails/EventSection';
import { classNames } from '@inngest/components/utils/classNames';
import { ulid } from 'ulid';

import SendEventButton from '@/components/Event/SendEventButton';
import { FunctionRunSection } from '@/components/Function/RunSection';
import { useSendEventMutation } from '@/store/devApi';
import { useEvent } from './useEvent';

export default function StreamDetails() {
  const params = useSearchParams();
  const eventID = params.get('event');
  const runID = params.get('run');

  const eventResult = useEvent(eventID ?? '', { skip: !eventID });
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
    <div
      className={classNames(
        'grid h-full text-white',

        // Need 2 columns if the run has an event.
        eventResult.data ? 'grid-cols-2' : 'grid-cols-1'
      )}
    >
      {eventResult.data && (
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
  );
}
