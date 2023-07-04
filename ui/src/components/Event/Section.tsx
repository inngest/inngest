'use client';

import { useMemo } from 'react';
import { usePrettyJson } from '../../hooks/usePrettyJson';
import { useSendEventMutation } from '../../store/devApi';
import { EventStatus, FunctionRunStatus, useGetEventQuery } from '../../store/generated';
import Button from '../Button';
import CodeBlock from '../CodeBlock';
import ContentCard from '../Content/ContentCard';
import FuncCard from '../Function/FuncCard';
import TimelineRow from '../Timeline/TimelineRow';
import TimelineStaticContent from '../Timeline/TimelineStaticContent';
import { usePathname } from 'next/navigation';

interface EventSectionProps {
  eventId: string;
}

export const EventSection = ({ eventId }: EventSectionProps) => {
  // const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetEventQuery({ id: eventId }, { pollingInterval: 1500 });
  const event = useMemo(() => query.data?.event, [query.data?.event]);
  const eventPayload = usePrettyJson(event?.raw);
  const pathname = usePathname();

  /**
   * Stop polling for changes when an event is in a final state.
   */
  // useEffect(() => {
  //   if (typeof event?.pendingRuns !== "number") return;
  //   setPollingInterval(event.pendingRuns > 0 ? 1000 : 0);
  // }, [event?.pendingRuns]);

  const [sendEvent] = useSendEventMutation();

  if (query.isLoading) {
    return <div>Loading...</div>;
  }

  if (!event) {
    return <div>Event not found</div>;
  }

  return (
    <ContentCard
      title={event.name || 'unknown'}
      date={event.createdAt}
      id={eventId}
      idPrefix={'Event ID'}
      active
      // button={<Button label="Open Event" icon={<IconFeed />} />}
    >
      {eventPayload ? (
        <div className="px-4 pt-4">
          <CodeBlock tabs={[{ label: 'Payload', content: eventPayload }]} />
        </div>
      ) : null}

      <div className="pr-4 pt-4">
        <TimelineRow status={EventStatus.Completed} iconOffset={0}>
          <TimelineStaticContent
            label="Event Received"
            date={event.createdAt}
            actionBtn={
              <>
                <Button label="Replay" btnAction={() => {}} />
                <Button label="Edit and replay" kind="secondary" btnAction={() => {}} />
              </>
            }
          />
        </TimelineRow>

        {event.functionRuns?.map((run, i, list) => {
          const status = run.waitingFor
            ? EventStatus.Paused
            : run.status || FunctionRunStatus.Completed;

          let ContextBar: React.ReactNode | undefined;

          if (run.waitingFor?.expiryTime) {
            if (run.waitingFor.eventName) {
              ContextBar = (
                <div className="flex-1">
                  <div className="flex flex-row justify-between items-center space-x-4">
                    <div>
                      Function waiting for <strong>{run.waitingFor.eventName}</strong> event
                      {run.waitingFor.expression ? ' matching the expression:' : ''}
                    </div>
                    {/* <div>Continue button</div> */}
                  </div>
                  <pre className="bg-slate-900 px-2 py-0.5 rounded border border-slate-700 mt-2">
                    {run.waitingFor.expression}
                  </pre>
                </div>
              );
            } else {
              ContextBar = (
                <div className="flex-1">
                  <div className="flex flex-row justify-between items-center">
                    <div>
                      Function paused for sleep until&nbsp;
                      <strong>{new Date(run.waitingFor.expiryTime).toLocaleTimeString()}</strong>
                    </div>
                    {/* <div>Continue button</div> */}
                  </div>
                </div>
              );
            }
          }

          return (
            <TimelineRow
              key={run.id}
              status={status}
              iconOffset={36}
              bottomLine={i < list.length - 1}
            >
              <FuncCard
                title={run.name || 'Unknown'}
                date={run.startedAt}
                id={run.id}
                status={status}
                active={pathname.includes(run.id)}
                badge={run.pendingSteps || 0}
                href={`/feed/events/${event.id}/runs/${run.id}`}
                contextualBar={ContextBar}
              />
            </TimelineRow>
          );
        })}
      </div>
    </ContentCard>
  );
};
