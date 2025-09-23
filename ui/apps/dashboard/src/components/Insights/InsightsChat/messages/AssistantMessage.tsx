'use client';

import React from 'react';
import type { TextUIPart } from '@inngest/use-agents';

type AssistantMessageProps = {
  part: TextUIPart;
};

export const AssistantMessage = ({ part }: AssistantMessageProps) => {
  return (
    <div className="text-text-basis inline-block max-w-[340px] whitespace-pre-wrap rounded-md px-0 py-1 text-sm">
      {part.content}
    </div>
  );
};
