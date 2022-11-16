import { historicEvents } from '../../../mock/historicEventFeed'
import TimelineRow from '../Timeline/TimelineRow'
import TimelineFeedContent from '../Timeline/TimelineFeedContent'
import statusStyles from '../../utils/statusStyles'

function ListItem({ datetime, id, badge, status }) {
  const eventStatusStyles = statusStyles(status)

  return (
    <button className="px-4 py-4 bg-transparent border-t border-slate-800/50 text-left group flex flex-col min-w-0 w-full first-of-type:border-transparent hover:bg-slate-800/40">
      <div className="flex items-start justify-between w-full">
        <div className="text-sm font-normal whitespace-nowrap overflow-hidden text-ellipsis grow pr-2 leading-none">
          <span className={`block text-xs ${eventStatusStyles.text}`}>
            {datetime}
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
  )
}

export default function HistoricalList() {
  return (
    <div className="flex flex-col overflow-y-scroll">
      {historicEvents.map((event, i) => (
        <ListItem
          key={i}
          datetime={event.datetime}
          id={event.id}
          badge={event.badge}
          status={event.status}
        />
      ))}
    </div>
  )
}
