import { useMemo, useState } from "preact/hooks";
import { useSendEventMutation } from "../../store/devApi";
import {
  EventStatus,
  FunctionRunStatus,
  useGetEventQuery,
} from "../../store/generated";
import { selectRun } from "../../store/global";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import Button from "../Button";
import ContentCard from "../Content/ContentCard";
import FuncCard from "../Function/FuncCard";
import TimelineRow from "../Timeline/TimelineRow";
import TimelineStaticContent from "../Timeline/TimelineStaticContent";
import { SendEventModal } from "./SendEventModal";

interface EventSectionProps {
  eventId: string;
}

export const EventSection = ({ eventId }: EventSectionProps) => {
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const dispatch = useAppDispatch();

  // const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetEventQuery({ id: eventId }, { pollingInterval: 1000 });
  const event = useMemo(() => query.data?.event, [query.data?.event]);

  /**
   * Stop polling for changes when an event is in a final state.
   */
  // useEffect(() => {
  //   if (typeof event?.pendingRuns !== "number") return;
  //   setPollingInterval(event.pendingRuns > 0 ? 1000 : 0);
  // }, [event?.pendingRuns]);

  const [eventModalVisible, setEventModalVisible] = useState(false);
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
      active
      // button={<Button label="Open Event" icon={<IconFeed />} />}
    >
      {eventModalVisible ? (
        <SendEventModal
          onClose={() => setEventModalVisible(false)}
          eventDataStr={event.raw}
        />
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

                    sendEvent(
                      JSON.stringify({
                        ...JSON.parse(event.raw),
                        ts: Date.now(),
                      })
                    );
                  }}
                />
                <Button
                  label="Edit and replay"
                  kind="secondary"
                  btnAction={() => {
                    setEventModalVisible((v) => !v);
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

          let contextBar;

          if (run.waitingFor?.waitUntil) {
            if (run.waitingFor.eventName) {
              contextBar = (
                <div className="flex-1">
                  <div className="flex flex-row justify-between items-center space-x-4">
                    <div>
                      Function waiting for{" "}
                      <strong>{run.waitingFor.eventName}</strong> event
                      {run.waitingFor.expression
                        ? "matching the expression"
                        : ""}
                    </div>
                    {/* <div>Continue button</div> */}
                  </div>
                  <pre>whassis</pre>
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
                          run.waitingFor.waitUntil
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
