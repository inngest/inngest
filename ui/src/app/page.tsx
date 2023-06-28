"use client";

import ActionBar from '@/components/ActionBar';
import BG from '@/components/BG';
import { BlankSlate } from '@/components/Blank';
import Button from '@/components/Button';
import ContentFrame from '@/components/Content/ContentFrame';
import { Docs } from '@/components/Docs';
import { EventSection } from '@/components/Event/Section';
import { SendEventModal } from '@/components/Event/SendEventModal';
import { EventStream } from '@/components/Event/Stream';
import { FunctionRunSection } from '@/components/Function/RunSection';
import { FuncStream } from '@/components/Function/Stream';
import Header from '@/components/Header';
import Navbar from '@/components/Navbar/Navbar';
import NavbarLink from '@/components/Navbar/NavbarLink';
import TimelineScrollContainer from '@/components/Timeline/TimelineScrollContainer';
import { IconBook, IconFeed, IconFunction } from '@/icons';
import { useGetEventsStreamQuery, useGetFunctionsStreamQuery } from '@/store/generated';
import {
  setSidebarTab,
  showDocs,
  showEventSendModal,
  showFeed,
  showFunctions,
} from '@/store/global';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import classNames from '@/utils/classnames';
import { FunctionList } from '@/views/FunctionList';

export default function Page() {
  const sidebarTab = useAppSelector((state) => state.global.sidebarTab);
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const contentView = useAppSelector((state) => state.global.contentView);
  const dispatch = useAppDispatch();

  const tabs: {
    key: typeof sidebarTab;
    title: string;
    onClick: () => void;
  }[] = [
    {
      key: "events",
      title: "Event Stream",
      onClick: () => dispatch(setSidebarTab("events")),
    },
    {
      key: "functions",
      title: "Function Log",
      onClick: () => dispatch(setSidebarTab("functions")),
    },
  ];

  const { hasEvents, isLoading: eventsLoading } = useGetEventsStreamQuery(
    undefined,
    {
      selectFromResult: (result) => ({
        ...result,
        hasEvents: Boolean(result.data?.events?.length || 0),
      }),
    }
  );

  const { hasRuns, isLoading: runsLoading } = useGetFunctionsStreamQuery(
    undefined,
    {
      selectFromResult: (result) => ({
        ...result,
        hasRuns: Boolean(result.data?.functionRuns?.length || 0),
      }),
    }
  );

  return (
    <div
      className={classNames(
        "w-screen h-screen text-slate-400 text-sm grid overflow-hidden relative",
        contentView === "feed"
          ? "grid-cols-app-sm xl:grid-cols-app 2xl:grid-cols-app-desktop grid-rows-app"
          : "grid-cols-docs grid-rows-docs"
      )}
    >
      <BG />
      {/* <EventDetail /> */}
      <Header>
        <Navbar>
          <NavbarLink
            icon={<IconFeed />}
            active={contentView === "feed"}
            badge={20}
            onClick={() => dispatch(showFeed())}
            tabName="Stream"
          />
          <NavbarLink
            icon={<IconFunction />}
            active={contentView === "functions"}
            onClick={() => dispatch(showFunctions())}
            tabName="Functions"
          />
          <NavbarLink
            icon={<IconBook />}
            active={contentView === "docs"}
            onClick={() => dispatch(showDocs())}
            tabName="Docs"
          />
        </Navbar>
      </Header>
      <SendEventModal />
      {contentView === "feed" ? (
        <>
          <ActionBar
            tabs={tabs.map((tab) => (
              <button
                key={tab.key}
                className={classNames(
                  sidebarTab === tab.key
                    ? `border-indigo-400 text-white`
                    : `border-transparent text-slate-400`,
                  `text-xs px-5 py-2.5 border-b block transition-all duration-150`
                )}
                onClick={tab.onClick}
              >
                {tab.title}
              </button>
            ))}
            actions={
              <Button
                label="Send event"
                btnAction={() => {
                  dispatch(
                    showEventSendModal({
                      show: true,
                      data: JSON.stringify({
                        name: "",
                        data: {},
                        user: {},
                      }),
                    })
                  );
                }}
              />
            }
          />
          <TimelineScrollContainer>
            {sidebarTab === "events" ? <EventStream /> : <FuncStream />}
          </TimelineScrollContainer>
          {selectedEvent ? (
            <ContentFrame>
              <EventSection eventId={selectedEvent} />
              <FunctionRunSection runId={selectedRun} />
            </ContentFrame>
          ) : eventsLoading || runsLoading ? null : sidebarTab === "events" ? (
            hasEvents ? (
              <BlankSlate
                title="No event selected"
                subtitle="Select an event from the stream on the left to view its details and which functions it's triggered."
                imageUrl="/images/no-fn-selected.png"
              />
            ) : (
              <BlankSlate
                title="Inngest hasn't received any events"
                subtitle="Read our documentation to learn how to send events to Inngest."
                imageUrl="/images/no-events.png"
                button={{
                  text: "Sending Events",
                  onClick: () => dispatch(showDocs("/events")),
                }}
              />
            )
          ) : hasRuns ? (
            <BlankSlate
              title="No run selected"
              subtitle="Select a function run from the stream on the left to view its details, trigger, and execution timeline."
              imageUrl="/images/no-fn-selected.png"
            />
          ) : (
            <BlankSlate
              title="No functions have been run yet"
              subtitle="We haven't run any functions in response to events or crons yet. Read our documentation to learn how to write and call a function."
              imageUrl="/images/no-results.png"
              button={{
                text: "Writing Functions",
                onClick: () => dispatch(showDocs("/functions")),
              }}
            />
          )}
        </>
      ) : contentView === "functions" ? (
        <FunctionList />
      ) : (
        <Docs />
      )}
    </div>
  );
}
