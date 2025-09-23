'use client';

import React from 'react';
import type { TextUIPart } from '@inngest/use-agents';

type AssistantMessageProps = {
  part: TextUIPart;
};

export const AssistantMessage = ({ part }: AssistantMessageProps) => {
  return (
    <div className="inline-block max-w-[340px] whitespace-pre-wrap rounded-md bg-indigo-100 px-3 py-2 text-sm text-gray-800">
      {part.content}
    </div>
  );
};
