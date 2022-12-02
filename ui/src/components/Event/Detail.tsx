import { useState } from "preact/hooks";
import classNames from "../../utils/classnames";

import TimelineFuncProgress from "../Timeline/TimelineFuncProgress";
import TimelineRow from "../Timeline/TimelineRow";
import TimelineStaticContent from "../Timeline/TimelineStaticContent";

import eventFuncs from "../../../mock/eventFuncs";
import { funcTabs } from "../../../mock/funcTabs";
import { eventTabs } from "../../../mock/tabs";
import { IconFeed } from "../../icons";
import { EventStatus, FunctionRunStatus } from "../../store/generated";
import Button from "../Button";
import CodeBlock from "../CodeBlock";
import ContentCard from "../Content/ContentCard";
import FuncCard from "../Function/FuncCard";
import HistoricalList from "./HistoricalList";

export default function EventDetail() {
  const [codeBlockModalActive, setCodeBlockModalActive] = useState({
    visible: false,
    content: "",
  });

  const [activeTab, setActiveTab] = useState(0);

  const setModal = (content) => {
    if (codeBlockModalActive.visible) {
      setCodeBlockModalActive({
        visible: false,
        content: "",
      });
    } else {
      setCodeBlockModalActive({
        visible: true,
        content: content,
      });
    }
  };

  const tabs = ["Event History", "Analytics"];

  return (
    // fake overlay
    <div className=" fixed flex inset-0 z-50 items-center justify-center bg-black/50 backdrop-blur-md py-12">
      <div className="grid grid-rows-event-overlay grid-cols-event-overlay bg-slate-1000 rounded overflow-hidden w-full max-h-full max-w-[1800px] mx-20">
        <header className="col-span-3 flex flex-col justify-between border-b border-slate-800/50 ">
          <div className=" h-full pt-4 px-4">
            <span className="mb-1 block">Event Overview</span>
            <h1 className="text-2xl text-slate-50 leading-none">
              accounts/profile
            </h1>
          </div>
          <div className="flex -mb-px">
            {tabs.map((tab, i) => (
              <button
                className={classNames(
                  i === activeTab
                    ? `border-indigo-400 text-white`
                    : `border-transparent text-slate-400`,
                  `text-xs px-5 py-2.5 border-b block transition-all duration-150`
                )}
                // onClick={() => handleTabClick(i)}
                key={i}
              >
                {tab}
              </button>
            ))}
          </div>
        </header>
        <div className="bg-slate-950">
          <h2 className="px-4 py-4 text-slate-400/80 bg-slate-1000/70 uppercase text-3xs">
            Previous Runs
          </h2>
          <HistoricalList />
        </div>
        <main className="flex flex-1 overflow-hidden row-start-2 col-start-3">
          <div className="flex gap-3 p-3 w-full min-w-0">
            <div className="flex-1 border rounded-lg border-slate-800/50 overflow-y-scroll flex flex-col shrink-0">
              <div className="pr-4 pt-4">
                <TimelineRow status={EventStatus.Completed} iconOffset={0}>
                  <TimelineStaticContent
                    label="Event Received"
                    date={"2022-04-28T14:34:21"}
                    actionBtn={<Button label="Replay" />}
                  />
                </TimelineRow>

                {eventFuncs.map((eventFunc, i) => {
                  return (
                    <TimelineRow
                      key={i}
                      status={eventFunc.status}
                      iconOffset={36}
                    >
                      <FuncCard
                        title={eventFunc.name}
                        date={eventFunc.datetime}
                        badge={eventFunc.version}
                        id={eventFunc.id}
                        // status={eventFunc.status}
                        status={FunctionRunStatus.Completed}
                        contextualBar={
                          <>
                            <p>Function paused for sleep until 1:40pm</p>
                            <Button label="Rerun" />
                          </>
                        }
                      />
                    </TimelineRow>
                  );
                })}

                <TimelineRow
                  status={EventStatus.Failed}
                  iconOffset={0}
                  bottomLine={false}
                >
                  <TimelineStaticContent label="Function 3 Errored with Error 404" />
                </TimelineRow>
              </div>
              <div className="border-t border-slate-800/50 m-4 mt-0 pt-4">
                <CodeBlock modal={setModal} tabs={eventTabs} />
              </div>
            </div>
            <ContentCard
              title="Process uploaded images"
              date={"2022-04-28T14:34:21"}
              button={<Button label="Open Function" icon={<IconFeed />} />}
              id="01GGG522ZATDGVQBCND4ZEAS6Z"
            >
              <div className="border-t border-slate-800/50 m-4 mt-0 pt-4">
                <CodeBlock modal={setModal} tabs={funcTabs} />
              </div>
              <div className="flex justify-end px-4 border-t border-slate-800/50 pt-4 mt-4">
                <Button label="Replay" />
              </div>
              <div className="pr-4 mt-4">
                <TimelineRow status={EventStatus.Completed} iconOffset={0}>
                  <TimelineFuncProgress
                    label="Function Started"
                    date={"2022-04-28T14:34:21"}
                    id="01GGG522ZATDGVQBCND4ZEAS6Z"
                  >
                    <CodeBlock modal={setModal} tabs={funcTabs} />
                  </TimelineFuncProgress>
                </TimelineRow>

                <TimelineRow status={EventStatus.Completed}>
                  <TimelineFuncProgress
                    label="Function Started"
                    date={"2022-04-28T14:34:21"}
                    id="01GGG522ZATDGVQBCND4ZEAS6Z"
                  />
                </TimelineRow>
                <TimelineRow status={EventStatus.Failed}>
                  <TimelineFuncProgress
                    label="Function Started"
                    date={"2022-04-28T14:34:21"}
                    id="01GGG522ZATDGVQBCND4ZEAS6Z"
                  />
                </TimelineRow>
                <TimelineRow status={EventStatus.Failed}>
                  <TimelineFuncProgress
                    label="Function Started"
                    date={"2022-04-28T14:34:21"}
                    id="01GGG522ZATDGVQBCND4ZEAS6Z"
                  />
                </TimelineRow>
                <TimelineRow status={EventStatus.Failed} bottomLine={false}>
                  <TimelineFuncProgress
                    label="Function Started"
                    date={"2022-04-28T14:34:21"}
                    id="01GGG522ZATDGVQBCND4ZEAS6Z"
                  />
                </TimelineRow>
              </div>
            </ContentCard>
          </div>
        </main>
      </div>
    </div>
  );
}
