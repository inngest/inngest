import eventStream from '../../mock/eventStream'
import EventFeed from './EventFeed'

export default function Main() {
  return <EventFeed items={eventStream} />
}
