import { useQuery } from "@apollo/client";
import { EVENTS_STREAM } from "../../coreapi";
import { EventStatus, GetEventsStreamQuery } from "../../gql/graphql";
import TimelineFeedContent from "../Timeline/TimelineFeedContent";
import TimelineRow from "../Timeline/TimelineRow";

export const EventStream = () => {
  const events = useQuery<GetEventsStreamQuery>(EVENTS_STREAM, {
    pollInterval: 1000,
  });

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
