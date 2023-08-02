import { EventStatus, useGetEventsStreamQuery } from '../../store/generated';
import { selectEvent } from '../../store/global';
import { useAppDispatch, useAppSelector } from '../../store/hooks';
import TimelineFeedContent from '../Timeline/TimelineFeedContent';
import TimelineRow from '../Timeline/TimelineRow';

export const EventStream = () => {
  const events = useGetEventsStreamQuery(undefined, { pollingInterval: 1500 });
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const dispatch = useAppDispatch();

  return (
    <>
      {events?.data?.events?.map((event, i, list) => (
        <TimelineRow
          key={event.id}
          status={event.status || EventStatus.Completed}
          iconOffset={30}
          topLine={i !== 0}
          bottomLine={i < list.length - 1}
        >
          <TimelineFeedContent
            date={event.createdAt}
            active={selectedEvent === event.id}
            status={event.status || EventStatus.Completed}
            badge={event.totalRuns || 0}
            name={event.name || 'Unknown'}
            onClick={() => dispatch(selectEvent(event.id))}
          />
        </TimelineRow>
      ))}
    </>
  );
};
