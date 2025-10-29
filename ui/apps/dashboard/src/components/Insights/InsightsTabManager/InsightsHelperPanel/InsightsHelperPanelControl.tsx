'use client';

import { useMemo, type ReactElement } from 'react';
import { Link } from '@inngest/components/Link/Link';
import { cn } from '@inngest/components/utils/classNames';

export interface HelperItem {
  action: () => void;
  href?: string;
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
      {items.map((item) => {
        const sharedProps = useMemo(
          () =>
            ({
              'aria-label': item.title,
              className: cn(
                'flex h-8 items-center justify-center rounded-md text-subtle transition-colors w-8',
                activeTitle !== item.title && 'hover:bg-canvasSubtle',
                activeTitle === item.title &&
                  'bg-secondary-4xSubtle hover:bg-secondary-3xSubtle text-info'
              ),
              title: item.title,
            } as const),
          [activeTitle, item.title]
        );

        if (item.href) {
          return (
            <Link
              {...sharedProps}
              key={item.title}
              href={item.href}
              rel="noopener noreferrer"
              target="_blank"
            >
              {item.icon}
            </Link>
          );
        }

        return (
          <button {...sharedProps} key={item.title} onClick={item.action} type="button">
            {item.icon}
          </button>
        );
      })}
    </div>
  );
}
