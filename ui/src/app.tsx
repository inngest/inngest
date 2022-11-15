import { useState } from 'preact/hooks'

import classNames from './utils/classnames'
import { feeds } from '../mock/eventStream'
import eventFuncs from '../mock/eventFuncs'
import './index.css'
import Header from './components/Header'

import Sidebar from './components/Sidebar/Sidebar'
import SidebarLink from './components/Sidebar/SidebarLink'

import ContentFrame from './components/Content/ContentFrame'
import ContentCard from './components/Content/ContentCard'

import TimelineRow from './components/Timeline/TimelineRow'
import TimelineScrollContainer from './components/Timeline/TimelineScrollContainer'
import TimelineFeedContent from './components/Timeline/TimelineFeedContent'
import TimelineStaticContent from './components/Timeline/TimelineStaticContent'
import TimelineFuncProgress from './components/Timeline/TimelineFuncProgress'

import Button from './components/Button'
import FuncCard from './components/Function/FuncCard'

import CodeBlock from './components/CodeBlock'
import CodeBlockModal from './components/CodeBlock/CodeBlockModal'

import { IconFeed, IconBook } from './icons'

import { eventTabs } from '../mock/tabs'
import { funcTabs } from '../mock/funcTabs'
import ActionBar from './components/ActionBar'

import EventDetail from './components/Event/Detail'

export function App() {
  const [codeBlockModalActive, setCodeBlockModalActive] = useState({
    visible: false,
    content: '',
  })

  const [activeFeed, setActiveFeed] = useState(1)

  const tabs = ['Event Stream', 'Function Log']

  const setModal = (content) => {
    if (codeBlockModalActive.visible) {
      setCodeBlockModalActive({
        visible: false,
        content: '',
      })
    } else {
      setCodeBlockModalActive({
        visible: true,
        content: content,
      })
    }
  }

  const handleTabClick = (index) => {
    setActiveFeed(index)
  }

  return (
    <div class="w-screen h-screen text-slate-400 text-sm grid grid-cols-app 2xl:grid-cols-app-desktop grid-rows-app overflow-hidden">
      {codeBlockModalActive.visible && (
        <CodeBlockModal closeModal={setModal}>
          <CodeBlock
            tabs={codeBlockModalActive.content}
            modal={setModal}
            expanded
          />
        </CodeBlockModal>
      )}
      {/* <EventDetail /> */}
      <Header />
      <Sidebar>
        <SidebarLink icon={<IconFeed />} active badge={20} />
        <SidebarLink icon={<IconBook />} />
      </Sidebar>
      <ActionBar
        tabs={feeds.map((tab, i) => (
          <button
            className={classNames(
              i === activeFeed
                ? `border-indigo-400 text-white`
                : `border-transparent text-slate-400`,
              `text-xs px-5 py-2.5 border-b block transition-all duration-150`
            )}
            onClick={() => handleTabClick(i)}
            key={i}
          >
            {tab.name}
          </button>
        ))}
      />
      <TimelineScrollContainer>
        {feeds[activeFeed].content.map((event, i) => (
          <TimelineRow key={i} status={event.status} iconOffset={30}>
            <TimelineFeedContent
              datetime={event.datetime}
              name={event.name}
              badge={event.badge}
              status={event.status}
            />
          </TimelineRow>
        ))}
      </TimelineScrollContainer>
      <ContentFrame>
        <ContentCard
          title="accounts/profile.photo.uploaded"
          datetime="14:34:21 28/04/2022"
          button={<Button label="Open Event" icon={<IconFeed />} />}
          id="01GGG522ZATDGVQBCND4ZEAS6Z"
          active
        >
          <div className="pr-4 pt-4">
            <TimelineRow status="COMPLETED" iconOffset={0}>
              <TimelineStaticContent
                label="Event Received"
                datetime="14:34:21 28/04/2022"
                actionBtn={<Button label="Retry" />}
              />
            </TimelineRow>

            {eventFuncs.map((eventFunc, i) => {
              return (
                <TimelineRow key={i} status={eventFunc.status} iconOffset={36}>
                  <FuncCard
                    title={eventFunc.name}
                    datetime={eventFunc.datetime}
                    badge={eventFunc.version}
                    id={eventFunc.id}
                    status={eventFunc.status}
                    actionBtn={<Button label="Rerun" />}
                  />
                </TimelineRow>
              )
            })}

            <TimelineRow status="FAILED" iconOffset={0} bottomLine={false}>
              <TimelineStaticContent label="Function 3 Errored with Error 404" />
            </TimelineRow>
          </div>
          <div className="border-t border-slate-800/50 m-4 mt-0 pt-4">
            <CodeBlock modal={setModal} tabs={eventTabs} />
          </div>
        </ContentCard>
        <ContentCard
          title="Process uploaded images"
          datetime="14:34:21 28/04/2022"
          button={<Button label="Open Function" icon={<IconFeed />} />}
          id="01GGG522ZATDGVQBCND4ZEAS6Z"
        >
          <div className="border-t border-slate-800/50 m-4 mt-0 pt-4">
            <CodeBlock modal={setModal} tabs={funcTabs} />
          </div>
          <div className="flex justify-end px-4 border-t border-slate-800/50 pt-4 mt-4">
            <Button label="Retry" />
          </div>
          <div className="pr-4 mt-4">
            <TimelineRow status="COMPLETED" iconOffset={0}>
              <TimelineFuncProgress
                label="Function Started"
                datetime="14:34:21 28/04/2022"
                id="01GGG522ZATDGVQBCND4ZEAS6Z"
              >
                <CodeBlock modal={setModal} tabs={funcTabs} />
              </TimelineFuncProgress>
            </TimelineRow>

            <TimelineRow status="COMPLETED">
              <TimelineFuncProgress
                label="Function Started"
                datetime="14:34:21 28/04/2022"
                id="01GGG522ZATDGVQBCND4ZEAS6Z"
              />
            </TimelineRow>
            <TimelineRow status="FAILED">
              <TimelineFuncProgress
                label="Function Started"
                datetime="14:34:21 28/04/2022"
                id="01GGG522ZATDGVQBCND4ZEAS6Z"
              />
            </TimelineRow>
            <TimelineRow status="FAILED">
              <TimelineFuncProgress
                label="Function Started"
                datetime="14:34:21 28/04/2022"
                id="01GGG522ZATDGVQBCND4ZEAS6Z"
              />
            </TimelineRow>
            <TimelineRow status="FAILED" bottomLine={false}>
              <TimelineFuncProgress
                label="Function Started"
                datetime="14:34:21 28/04/2022"
                id="01GGG522ZATDGVQBCND4ZEAS6Z"
              />
            </TimelineRow>
          </div>
        </ContentCard>
      </ContentFrame>
    </div>
  )
}
