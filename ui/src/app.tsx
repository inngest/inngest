import './index.css'
import Header from './components/Header'
import Sidebar from './components/Sidebar/Sidebar'
import SidebarLink from './components/Sidebar/SidebarLink'
import ContentFrame from './components/Content/ContentFrame'
import eventStream from '../mock/eventStream'
import TimelineItem from './components/Timeline/TimelineItem'
import TimelineScrollContainer from './components/Timeline/TimelineScrollContainer'
import TimelineFeedContent from './components/Timeline/TimelineFeedContent'
import ContentCard from './components/Content/ContentCard'
import EventCard from './components/Event/EventCard'
import Button from './components/Button'

import { IconFeed, IconBook } from './icons'

export function App() {
  return (
    <div class="w-screen h-screen text-slate-400 text-sm grid grid-cols-app grid-rows-app overflow-hidden">
      <Header />
      <Sidebar>
        <SidebarLink icon={<IconFeed />} active badge={20} />
        <SidebarLink icon={<IconBook />} />
      </Sidebar>
      <ContentFrame>
        <TimelineScrollContainer>
          {eventStream.map((event, i) => (
            <TimelineItem key={i} status={event.status}>
              <TimelineFeedContent
                datetime={event.datetime}
                name={event.name}
                badge={event.fnCount}
                status={event.status}
              />
            </TimelineItem>
          ))}
        </TimelineScrollContainer>
        <div className="flex gap-3 p-3 w-full">
          <ContentCard
            title="accounts/profile.photo.uploaded"
            datetime="14:34:21 28/04/2022"
            button={<Button label="Open Event" icon={<IconFeed />} />}
            id="01GGG522ZATDGVQBCND4ZEAS6Z"
            active
          >
            <EventCard />
          </ContentCard>
          <ContentCard>
            <EventCard />
          </ContentCard>
        </div>
      </ContentFrame>
    </div>
  )
}
