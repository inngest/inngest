import { useSearchParams } from 'next/navigation';

import { EventSection } from '@/components/Event/Section';
import { FunctionRunSection } from '@/components/Function/RunSection';

export default function StreamDetails() {
  const params = useSearchParams();
  const eventID = params.get('event');
  const cronID = params.get('cron');
  const runID = params.get('run');

  return (
    <>
      {eventID && (
        <div className="grid grid-cols-2 h-full text-white">
          <EventSection eventId={eventID} />
          <FunctionRunSection runId={runID} />
        </div>
      )}
      {cronID && runID && (
        <div className="grid grid-cols-1 h-full text-white">
          <FunctionRunSection runId={runID} />
        </div>
      )}
    </>
  );
}
