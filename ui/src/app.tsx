import eventStream from '../mock/eventStream'
import eventFuncs from '../mock/eventFuncs'
import './index.css'
import Header from './components/Header'
import Sidebar from './components/Sidebar/Sidebar'
import SidebarLink from './components/Sidebar/SidebarLink'
import ContentFrame from './components/Content/ContentFrame'
import TimelineItem from './components/Timeline/TimelineItem'
import TimelineScrollContainer from './components/Timeline/TimelineScrollContainer'
import TimelineFeedContent from './components/Timeline/TimelineFeedContent'
import ContentCard from './components/Content/ContentCard'
import Button from './components/Button'
import FuncCard from './components/Function/FuncCard'
import CodeBlock from './components/CodeBlock'

import { IconFeed, IconBook } from './icons'
import TimelineStaticRow from './components/Timeline/TimelineStaticRow'

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
        <div className="flex gap-3 p-3 w-full min-w-0">
          <ContentCard
            title="accounts/profile.photo.uploaded"
            datetime="14:34:21 28/04/2022"
            button={<Button label="Open Event" icon={<IconFeed />} />}
            id="01GGG522ZATDGVQBCND4ZEAS6Z"
            active
          >
            <div className="">
              <TimelineItem status="COMPLETED">
                <TimelineStaticRow
                  label="Event Received"
                  datetime="14:34:21 28/04/2022"
                  actionBtn={<Button label="Retry" />}
                />
              </TimelineItem>

              {eventFuncs.map((eventFunc, i) => {
                return (
                  <TimelineItem key={i} status={eventFunc.status}>
                    <FuncCard
                      title={eventFunc.name}
                      datetime={eventFunc.datetime}
                      badge={eventFunc.version}
                      id={eventFunc.id}
                      status={eventFunc.status}
                      actionBtn={<Button label="Rerun" />}
                    />
                  </TimelineItem>
                )
              })}

              <TimelineItem status="FAILED">
                <TimelineStaticRow label="Function 3 Errored with Error 404" />
              </TimelineItem>
            </div>
            <div className="border-t border-slate-800/50 m-4 mt-0 pt-4">
              <CodeBlock />
            </div>
          </ContentCard>
          <ContentCard
            title="Process uploaded images"
            datetime="14:34:21 28/04/2022"
            button={<Button label="Open Function" icon={<IconFeed />} />}
            id="01GGG522ZATDGVQBCND4ZEAS6Z"
          >
            <h1>Function Content</h1>
          </ContentCard>
        </div>
      </ContentFrame>
    </div>
  )
}
