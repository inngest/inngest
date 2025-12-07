import React from "react";

import ClearThreadButton from "./ClearThreadButton";
import { ToggleChatButton } from "./ToggleChatButton";

type ChatHeaderProps = {
  onClearThread: () => void;
  onToggleChat: () => void;
};

export function ChatHeader({ onClearThread, onToggleChat }: ChatHeaderProps) {
  return (
    <div className="border-subtle bg-surfaceBase flex items-center justify-between border-b px-4 py-3">
      <div className="flex items-center gap-3">
        <ToggleChatButton onClick={onToggleChat} />
        <p className="text-basis text-sm">Insights AI</p>
      </div>
      <ClearThreadButton onClick={onClearThread} />
    </div>
  );
}
