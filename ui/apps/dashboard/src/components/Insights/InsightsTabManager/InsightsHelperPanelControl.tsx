'use client';

import type { ReactElement } from 'react';
import { cn } from '@inngest/components/utils/classNames';

export interface HelperItem {
  action: () => void;
  icon: ReactElement;
  title: string;
}

type InsightsHelperPanelControlProps = {
  activeTitle: null | string;
  items: HelperItem[];
};

export function InsightsHelperPanelControl({
  activeTitle,
  items,
}: InsightsHelperPanelControlProps) {
  return (
    <div className="border-subtle flex h-full w-[56px] flex-col items-center gap-2 border-l px-3 py-2">
      {items.map((item) => (
        <button
          key={item.title}
          aria-label={item.title}
          className={cn(
            'text-subtle hover:text-basis flex h-8 w-8 items-center justify-center rounded-md transition-colors',
            activeTitle === item.title && 'bg-secondary-4xSubtle text-info'
          )}
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
