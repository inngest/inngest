'use client';

import { useState } from 'react';

import ActionBar from '@/components/ActionBar';
import { BlankSlate } from '@/components/Blank';
import ContentFrame from '@/components/Content/ContentFrame';
import { EventSection } from '@/components/Event/Section';
import SendEventButton from '@/components/Event/SendEventButton';
import { EventStream } from '@/components/Event/Stream';
import { FunctionRunSection } from '@/components/Function/RunSection';
import { FuncStream } from '@/components/Function/Stream';
import TimelineScrollContainer from '@/components/Timeline/TimelineScrollContainer';
import { useGetEventsStreamQuery, useGetFunctionsStreamQuery } from '@/store/generated';
import { useAppSelector } from '@/store/hooks';
import classNames from '@/utils/classnames';

export default function Stream() {
  const [secondaryTab, setSecondaryTab] = useState('events');
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const selectedRun = useAppSelector((state) => state.global.selectedRun);

  const { hasEvents, isLoading: eventsLoading } = useGetEventsStreamQuery(undefined, {
    selectFromResult: (result) => ({
      ...result,
      hasEvents: Boolean(result.data?.events?.length || 0),
    }),
  });

  const { hasRuns, isLoading: runsLoading } = useGetFunctionsStreamQuery(undefined, {
    selectFromResult: (result) => ({
      ...result,
      hasRuns: Boolean(result.data?.functionRuns?.length || 0),
    }),
  });

  const tabs: {
    key: typeof secondaryTab;
    title: string;
    onClick: () => void;
  }[] = [
    {
      key: 'events',
      title: 'Event Stream',
      onClick: () => {
        setSecondaryTab('events');
      },
    },
    {
      key: 'functions',
      title: 'Function Log',
      onClick: () => {
        setSecondaryTab('functions');
      },
    },
  ];

  return (
    <>
      <ActionBar
        tabs={tabs.map((tab) => (
          <button
            key={tab.key}
            className={classNames(
              secondaryTab === tab.key
                ? `border-indigo-400 text-white`
                : `border-transparent text-slate-400`,
              `text-xs px-5 py-2.5 border-b block transition-all duration-150`,
            )}
            onClick={tab.onClick}
          >
            {tab.title}
          </button>
        ))}
        actions={
          <SendEventButton
            label="Send event"
            data={JSON.stringify({
              name: '',
              data: {},
              user: {},
            })}
          />
        }
      />
      <TimelineScrollContainer>
        {secondaryTab === 'events' ? <EventStream /> : <FuncStream />}
      </TimelineScrollContainer>
      {selectedEvent ? (
        <ContentFrame>
          <EventSection eventId={selectedEvent} />
          <FunctionRunSection runId={selectedRun} />
        </ContentFrame>
      ) : eventsLoading || runsLoading ? null : secondaryTab === 'events' ? (
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
            link={{
              text: 'Sending Events',
              url: 'https://www.inngest.com/docs/events',
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
          link={{
            text: 'Writing Functions',
            url: 'https://www.inngest.com/docs/functions',
          }}
        />
      )}
    </>
  );
}
