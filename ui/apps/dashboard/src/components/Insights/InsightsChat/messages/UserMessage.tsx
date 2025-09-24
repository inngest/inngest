'use client';

import type { MessagePart } from '@inngest/use-agent';

export const UserMessage = ({ part }: { part: MessagePart }) => {
  if (part.type !== 'text') {
    return null;
  }

  return (
    <div className="group relative flex justify-end">
      <div className="text-text-basis mb-2 inline-block max-w-[340px] whitespace-pre-wrap rounded-md bg-gray-100 px-3 py-2 text-sm dark:bg-[#353535]">
        {part.content}
      </div>
      {/* <ActionBar text={part.content} /> */}
    </div>
  );
};
