import React from 'react';

import ClearThreadButton from './ClearThreadButton';
import { ToggleChatButton } from './ToggleChatButton';

type ChatHeaderProps = {
  onClearThread: () => void;
  onToggleChat: () => void;
};

export function ChatHeader({ onClearThread, onToggleChat }: ChatHeaderProps) {
  return (
    <div className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3 dark:bg-zinc-900">
      <div className="flex items-center gap-3">
        <ClearThreadButton onClick={onClearThread} />
        <ToggleChatButton onClick={onToggleChat} />
      </div>
    </div>
  );
}
