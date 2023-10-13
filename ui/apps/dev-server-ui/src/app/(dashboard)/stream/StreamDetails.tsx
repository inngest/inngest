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
  const event = eventResult.data;

  const [selectedRunID, setSelectedRunID] = useState<string | undefined>(runID ?? undefined);
  const [sendEvent] = useSendEventMutation();

  useEffect(() => {
    if (!selectedRunID && event?.functionRuns[0]) {
      const firstRunID = event.functionRuns[0].id;
      setSelectedRunID(firstRunID);
    }
  }, [selectedRunID, event?.functionRuns]);

  function onReplayEvent() {
    if (!event?.payload) {
      return;
    }

    const eventId = ulid();

    sendEvent({
      ...JSON.parse(event.payload),
      id: eventId,
      ts: Date.now(),
    }).unwrap();
  }

  const renderSendEventButton = useMemo(() => {
    return () => (
      <SendEventButton label="Edit and Replay" appearance="outlined" data={event?.payload} />
    );
  }, [event?.payload]);

  return (
    <div
      className={classNames(
        'grid h-full text-white',

        // Need 2 columns if the run has an event.
        event ? 'grid-cols-2' : 'grid-cols-1'
      )}
    >
      {event && (
        <EventSection
          event={event}
          functionRuns={event.functionRuns}
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
