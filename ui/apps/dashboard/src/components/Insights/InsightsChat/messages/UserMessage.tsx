'use client';

import type { MessagePart } from '@inngest/use-agent';

export const UserMessage = ({ part }: { part: MessagePart }) => {
  if (part.type !== 'text') {
    return null;
  }

  return (
    <div className="group relative flex justify-end">
      <div className="text-basis bg-canvasSubtle mb-2 inline-block max-w-[340px] whitespace-pre-wrap rounded-lg px-3 py-2 text-start text-sm shadow-[inset_0_-1px_3px_0_rgb(var(--color-foreground-base)/0.08)]">
        {part.content}
      </div>
    </div>
  );
};
