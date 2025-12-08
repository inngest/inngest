"use client";

import { TextCell } from "@inngest/components/Table";

import { getFormattedJSONObjectOrArrayString } from "./json";

interface JSONAwareTextCellProps {
  children: string;
}

export function JSONAwareTextCell({ children }: JSONAwareTextCellProps) {
  const formattedJSON = getFormattedJSONObjectOrArrayString(children);
  if (formattedJSON === null) return <TextCell>{children}</TextCell>;

  return (
    <div className="text-basis text-sm font-medium">
      <pre
        className="max-h-[160px] max-w-none overflow-hidden outline-none [scrollbar-width:thin] group-focus-within:overflow-auto [&::-webkit-scrollbar]:w-1"
        tabIndex={0}
      >
        {formattedJSON}
      </pre>
    </div>
  );
}
