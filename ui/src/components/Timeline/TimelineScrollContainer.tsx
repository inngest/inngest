import FeedItem from './FeedItem'
import EventFeedItem from './EventFeedItem'

export default function TimelineContainer({ children }) {
  return (
    <ul className="bg-slate-950 border-r border-slate-800 basis-[340px] overflow-y-scroll relative py-4 shrink-0">
      {children}
    </ul>
  )
}
