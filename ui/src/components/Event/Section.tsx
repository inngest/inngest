import { useEffect, useMemo, useState } from "preact/hooks";
import { IconFeed } from "../../icons";
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

interface EventSectionProps {
  eventId: string;
}

export const EventSection = ({ eventId }: EventSectionProps) => {
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const dispatch = useAppDispatch();

  const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetEventQuery({ id: eventId }, { pollingInterval });
  const event = useMemo(() => query.data?.event, [query.data?.event]);

  /**
   * Stop polling for changes when an event is in a final state.
   */
  useEffect(() => {
    if (typeof event?.pendingRuns !== "number") return;
    setPollingInterval(event.pendingRuns > 0 ? 1000 : 0);
  }, [event?.pendingRuns]);

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
      button={<Button label="Open Event" icon={<IconFeed />} />}
    >
      <div className="pr-4 pt-4">
        <TimelineRow status={EventStatus.Completed} iconOffset={0}>
          <TimelineStaticContent
            label="Event Received"
            date={event.createdAt}
            actionBtn={<Button label="Retry" />}
          />
        </TimelineRow>

        {event.functionRuns?.map((run, i, list) => (
          <TimelineRow
            key={run.id}
            status={run.status || FunctionRunStatus.Completed}
            iconOffset={36}
            bottomLine={i < list.length - 1}
          >
            <FuncCard
              title={run.name || "Unknown"}
              date={run.startedAt}
              id={run.id}
              status={run.status || FunctionRunStatus.Completed}
              active={selectedRun === run.id}
              badge={run.pendingSteps || 0}
              onClick={() => dispatch(selectRun(run.id))}
            />
          </TimelineRow>
        ))}
      </div>
    </ContentCard>
  );
};
