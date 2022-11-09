import eventFuncs from '../../../mock/eventFuncs'
import TimelineItem from '../Timeline/TimelineItem'
import FuncCard from '../Function/FuncCard'

export default function EventCard({ children }) {
  return (
    <div className="pt-3.5 flex flex-col">
      {eventFuncs.map((eventFunc, i) => {
        return (
          <TimelineItem key={i} status={eventFunc.status}>
            <FuncCard
              title={eventFunc.name}
              datetime={eventFunc.datetime}
              badge={eventFunc.version}
              id={eventFunc.id}
              status={eventFunc.status}
            />
          </TimelineItem>
        )
      })}
    </div>
  )
}
