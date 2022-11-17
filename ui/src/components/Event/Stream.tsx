import { EventStatus, useGetEventsStreamQuery } from "../../store/generated";
import TimelineFeedContent from "../Timeline/TimelineFeedContent";
import TimelineRow from "../Timeline/TimelineRow";

export const EventStream = () => {
  const events = useGetEventsStreamQuery(
    {},
    {
      pollingInterval: 1000,
    }
  );

  return (
    <>
      {events.data?.events?.map((event, i) => (
        <TimelineRow
          key={event.id}
          status={event.status || EventStatus.Completed}
          iconOffset={30}
        >
          <TimelineFeedContent
            date={new Date(event.createdAt)}
            status={event.status || EventStatus.Completed}
            badge={event.pendingRuns || 0}
            name={event.name || "unknown"}
          />
        </TimelineRow>
      ))}
    </>
  );
};
