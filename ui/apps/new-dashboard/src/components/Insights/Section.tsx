"use client";

import type { ReactNode } from "react";
import { cn } from "@inngest/components/utils/classNames";

export interface SectionProps {
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  title: ReactNode;
}

export function Section({ actions, children, className, title }: SectionProps) {
  return (
    <div
      className={cn(
        "flex h-full w-full min-w-0 flex-col overflow-hidden",
        className,
      )}
    >
      <header className="border-subtle flex w-full items-center justify-between overflow-x-auto border-b px-4 py-2">
        <div className="text-basis flex-shrink-0 text-xs font-medium tracking-wide">
          {title}
        </div>
        {actions && (
          <div className="flex shrink-0 items-center gap-2">{actions}</div>
        )}
      </header>
      <div className="w-full min-w-0 flex-1 overflow-hidden">{children}</div>
    </div>
  );
}
