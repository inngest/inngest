import { useMemo } from "react";
import { ulid } from 'ulid';
import { usePrettyJson } from "../../hooks/usePrettyJson";
import { useSendEventMutation } from "../../store/devApi";
import {
  EventStatus,
  FunctionRunStatus,
  useGetEventQuery,
} from "../../store/generated";
import { selectEvent, selectRun, showEventSendModal } from "../../store/global";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import Button from "../Button";
import CodeBlock from "../CodeBlock";
import ContentCard from "../Content/ContentCard";
import FuncCard from "../Function/FuncCard";
import TimelineRow from "../Timeline/TimelineRow";
import TimelineStaticContent from "../Timeline/TimelineStaticContent";

interface EventSectionProps {
  eventId: string;
}

export const EventSection = ({ eventId }: EventSectionProps) => {
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const dispatch = useAppDispatch();

  // const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetEventQuery({ id: eventId }, { pollingInterval: 1500 });
  const event = useMemo(() => query.data?.event, [query.data?.event]);
  const eventPayload = usePrettyJson(event?.raw);

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
      title={event.name || "unknown"}
      date={event.createdAt}
      id={eventId}
      idPrefix={"Event ID"}
      active
      // button={<Button label="Open Event" icon={<IconFeed />} />}
    >
      {eventPayload ? (
        <div className="px-4 pt-4">
          <CodeBlock tabs={[{ label: "Payload", content: eventPayload }]}/>
        </div>
      ) : null}
      <div className="pr-4 pt-4">
        <TimelineRow status={EventStatus.Completed} iconOffset={0}>
          <TimelineStaticContent
            label="Event Received"
            date={event.createdAt}
            actionBtn={
              <>
                <Button
                  label="Replay"
                  btnAction={() => {
                    if (!event?.raw) {
                      return;
                    }

                    const eventId = ulid();

                    sendEvent(
                      {
                        ...JSON.parse(event.raw),
                        id: eventId,
                        ts: Date.now(),
                      },
                    ).unwrap().then(() => {
                      dispatch(selectEvent(eventId));
                    })
                  }}
                />
                <Button
                  label="Edit and replay"
                  kind="secondary"
                  btnAction={() => {
                    dispatch(
                      showEventSendModal({ show: true, data: event.raw })
                    );
                  }}
                />
              </>
            }
          />
        </TimelineRow>

        {event.functionRuns?.map((run, i, list) => {
          const status = run.waitingFor
            ? EventStatus.Paused
            : run.status || FunctionRunStatus.Completed;

          let contextBar: React.ReactNode | undefined;

          if (run.waitingFor?.expiryTime) {
            if (run.waitingFor.eventName) {
              contextBar = (
                <div className="flex-1">
                  <div className="flex flex-row justify-between items-center space-x-4">
                    <div>
                      Function waiting for{" "}
                      <strong>{run.waitingFor.eventName}</strong> event
                      {run.waitingFor.expression
                        ? " matching the expression:"
                        : ""}
                    </div>
                    {/* <div>Continue button</div> */}
                  </div>
                  <pre className="bg-slate-900 px-2 py-0.5 rounded border border-slate-700 mt-2">
                    {run.waitingFor.expression}
                  </pre>
                </div>
              );
            } else {
              contextBar = (
                <div className="flex-1">
                  <div className="flex flex-row justify-between items-center">
                    <div>
                      Function paused for sleep until&nbsp;
                      <strong>
                        {new Date(
                          run.waitingFor.expiryTime
                        ).toLocaleTimeString()}
                      </strong>
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
                title={run.name || "Unknown"}
                date={run.startedAt}
                id={run.id}
                status={status}
                active={selectedRun === run.id}
                badge={run.pendingSteps || 0}
                onClick={() => dispatch(selectRun(run.id))}
                contextualBar={contextBar}
              />
            </TimelineRow>
          );
        })}
      </div>
    </ContentCard>
  );
};
