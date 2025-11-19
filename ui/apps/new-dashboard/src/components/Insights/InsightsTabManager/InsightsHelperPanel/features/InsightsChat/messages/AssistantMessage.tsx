"use client";

import React from "react";
import type { TextUIPart } from "@inngest/use-agent";

type AssistantMessageProps = {
  part: TextUIPart;
};

export const AssistantMessage = ({ part }: AssistantMessageProps) => {
  return (
    <div className="text-basis inline-block max-w-full whitespace-pre-wrap rounded-md px-0 py-1 text-sm">
      {part.content}
    </div>
  );
};
