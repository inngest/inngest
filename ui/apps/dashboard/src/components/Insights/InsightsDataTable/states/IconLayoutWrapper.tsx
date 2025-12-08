"use client";

import { RiTableView } from "@remixicon/react";

type IconLayoutWrapperProps = {
  action: React.ReactNode;
  header: string;
  subheader: string;
};

export function IconLayoutWrapper({
  action,
  header,
  subheader,
}: IconLayoutWrapperProps) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-4">
      <div className="flex max-w-[410px] flex-col items-center gap-4">
        <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
          <RiTableView className="text-light h-6 w-6" />
        </div>
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-basis text-xl font-medium">{header}</h3>
          <p className="text-muted text-sm">{subheader}</p>
        </div>
        {action}
      </div>
    </div>
  );
}
