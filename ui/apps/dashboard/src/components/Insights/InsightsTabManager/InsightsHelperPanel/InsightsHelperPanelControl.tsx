import { type ReactElement } from 'react';
import { Link } from '@inngest/components/Link';
import { cn } from '@inngest/components/utils/classNames';

export interface HelperItem {
  action: () => void;
  href?: string;
  icon: ReactElement;
  label?: string;
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
    <div className="border-subtle flex h-full w-[70px] flex-col items-center gap-2 border-l px-2 py-2">
      {items.map((item) => {
        const sharedProps = {
          'aria-label': item.title,
          className: cn(
            'flex w-full flex-col items-center justify-center gap-0.5 rounded-md py-1 text-subtle transition-colors',
            activeTitle !== item.title && 'hover:bg-canvasSubtle',
            activeTitle === item.title &&
              'bg-secondary-4xSubtle hover:bg-secondary-3xSubtle text-info',
          ),
          title: item.title,
        } as const;

        const content = (
          <>
            <div className="flex h-8 w-8 items-center justify-center">
              {item.icon}
            </div>
            <div className="max-w-full px-1 text-center font-medium text-[9px] leading-tight text-subtle">
              <span className="block whitespace-normal break-words">
                {item.label ?? item.title}
              </span>
            </div>
          </>
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
              {content}
            </Link>
          );
        }

        return (
          <button
            {...sharedProps}
            key={item.title}
            onClick={item.action}
            type="button"
          >
            {content}
          </button>
        );
      })}
    </div>
  );
}
