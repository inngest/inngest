import { useState } from "preact/hooks";

import BG from "./components/BG";

import Header from "./components/Header";
import "./index.css";

import Sidebar from "./components/Sidebar/Sidebar";
import SidebarLink from "./components/Sidebar/SidebarLink";

import ContentFrame from "./components/Content/ContentFrame";

import TimelineScrollContainer from "./components/Timeline/TimelineScrollContainer";

import CodeBlock from "./components/CodeBlock";
import CodeBlockModal from "./components/CodeBlock/CodeBlockModal";

import { IconBook, IconFeed } from "./icons";

import ActionBar from "./components/ActionBar";
import { EventSection } from "./components/Event/Section";
import { EventStream } from "./components/Event/Stream";
import { FunctionRunSection } from "./components/Function/RunSection";
import { FuncStream } from "./components/Function/Stream";
import { setSidebarTab } from "./store/global";
import { useAppDispatch, useAppSelector } from "./store/hooks";
import classNames from "./utils/classnames";

export function App() {
  const sidebarTab = useAppSelector((state) => state.global.sidebarTab);
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
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
    <div class="w-screen h-screen text-slate-400 text-sm grid grid-cols-app-sm xl:grid-cols-app 2xl:grid-cols-app-desktop grid-rows-app overflow-hidden">
      <BG />
      {codeBlockModalActive.visible && (
        <CodeBlockModal closeModal={setModal}>
          <CodeBlock
            tabs={[{ label: "Payload", content: codeBlockModalActive.content }]}
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
    </div>
  );
}
