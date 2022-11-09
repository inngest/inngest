import EventFeedItem from './EventFeedItem'

export default function EventFeed({ items }) {
  return (
    <div className="bg-slate-950 border-r border-slate-800 w-[340px] overflow-y-scroll relative py-4">
      <ul>
        {items.map((event, i) => (
          <EventFeedItem
            key={i}
            datetime={event.datetime}
            status={event.status}
            name={event.name}
            fnCount={event.fnCount}
          />
        ))}
      </ul>
    </div>
  )
}
