'use client';

import type { ReactElement } from 'react';

export interface HelperItem {
  action: () => void;
  icon: ReactElement;
  title: string;
}

type InsightsHelperPanelCollapsedProps = {
  items: HelperItem[];
};

export function InsightsHelperPanelCollapsed({ items }: InsightsHelperPanelCollapsedProps) {
  return (
    <div className="border-subtle flex h-full w-[56px] flex-col items-center gap-2 border-l px-3 py-2">
      {items.map((item) => (
        <button
          key={item.title}
          aria-label={item.title}
          className="text-subtle hover:bg-canvasSubtle hover:text-basis flex h-8 w-8 items-center justify-center rounded-md transition-colors"
          onClick={() => item.action()}
          title={item.title}
          type="button"
        >
          {item.icon}
        </button>
      ))}
    </div>
  );
}
