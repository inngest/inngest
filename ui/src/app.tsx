import { useState } from "preact/hooks";
import ActionBar from "./components/ActionBar";
import BG from "./components/BG";
import ContentFrame from "./components/Content/ContentFrame";
import { Docs } from "./components/Docs";
import { EventSection } from "./components/Event/Section";
import { EventStream } from "./components/Event/Stream";
import { FunctionRunSection } from "./components/Function/RunSection";
import { FuncStream } from "./components/Function/Stream";
import Header from "./components/Header";
import Sidebar from "./components/Sidebar/Sidebar";
import SidebarLink from "./components/Sidebar/SidebarLink";
import TimelineScrollContainer from "./components/Timeline/TimelineScrollContainer";
import { IconBook, IconFeed } from "./icons";
import "./index.css";
import { selectContentView, setSidebarTab } from "./store/global";
import { useAppDispatch, useAppSelector } from "./store/hooks";
import classNames from "./utils/classnames";

export function App() {
  const sidebarTab = useAppSelector((state) => state.global.sidebarTab);
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const contentView = useAppSelector((state) => state.global.contentView);
  const dispatch = useAppDispatch();

  const [codeBlockModalActive, setCodeBlockModalActive] = useState({
    visible: false,
    content: "",
  });

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

  return (
    <div
      class={classNames(
        "w-screen h-screen text-slate-400 text-sm grid overflow-hidden relative",
        contentView === "feed"
          ? "grid-cols-app-sm xl:grid-cols-app 2xl:grid-cols-app-desktop grid-rows-app"
          : "grid-cols-docs grid-rows-docs"
      )}
    >
      <BG />
      {/* <EventDetail /> */}
      <Header />
      <Sidebar>
        <SidebarLink
          icon={<IconFeed />}
          active={contentView === "feed"}
          badge={20}
          onClick={() => dispatch(selectContentView("feed"))}
        />
        <SidebarLink
          icon={<IconBook />}
          active={contentView === "docs"}
          onClick={() => dispatch(selectContentView("docs"))}
        />
      </Sidebar>
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
          />
          <TimelineScrollContainer>
            {sidebarTab === "events" ? <EventStream /> : <FuncStream />}
          </TimelineScrollContainer>
          {selectedEvent ? (
            <ContentFrame>
              <EventSection eventId={selectedEvent} />
              <FunctionRunSection runId={selectedRun} />
            </ContentFrame>
          ) : null}
        </>
      ) : (
        <Docs />
      )}
    </div>
  );
}
