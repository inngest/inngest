import { EventStatus, useGetEventsStreamQuery } from "../../store/generated";
import statusStyles from "../../utils/statusStyles";

interface ListItemProps {
  date: Date;
  id: string;
  status: EventStatus;
  badge: number;
}

function ListItem({ date, id, badge, status }: ListItemProps) {
  const eventStatusStyles = statusStyles(status);

  return (
    <button className="px-4 py-4 bg-transparent border-t border-slate-800/50 text-left group flex flex-col min-w-0 w-full first-of-type:border-transparent hover:bg-slate-800/40">
      <div className="flex items-start justify-between w-full">
        <div className="text-sm font-normal whitespace-nowrap overflow-hidden text-ellipsis grow pr-2 leading-none">
          <span className={`block text-xs ${eventStatusStyles.text}`}>
            {date.toISOString()}
          </span>
          <span className={`block text-3xs text-slate-500 mt-2`}>{id}</span>
        </div>
        <span
          className={`rounded-md ${eventStatusStyles.fnBG} text-slate-100 text-3xs font-semibold leading-none flex items-center justify-center py-1.5 px-2`}
        >
          {badge}
        </span>
      </div>
    </button>
  );
}

export default function HistoricalList() {
  const events = useGetEventsStreamQuery({}, { pollingInterval: 1000 });

  return (
    <div className="flex flex-col overflow-y-scroll">
      {events.data?.events?.map((event, i) => (
        <ListItem
          key={event.id}
          date={new Date(event.createdAt)}
          id={event.id}
          badge={event.pendingRuns || 0}
          status={event.status || EventStatus.Completed}
        />
      ))}
    </div>
  );
}
