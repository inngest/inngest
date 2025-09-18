'use client';

import type { MessagePart, TextUIPart } from '@inngest/use-agents';

export const AssistantMessage = ({ part }: { part: MessagePart }) => {
  if (part.type !== 'text') {
    return null;
  }
  return (
    <div className="text-text-basis inline-block max-w-[340px] whitespace-pre-wrap rounded-md px-0 py-1 text-sm">
      {(part as TextUIPart).content}
    </div>
  );
};
