import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { cn } from '@inngest/components/utils/classNames';

interface ContentCardProps {
  children: React.ReactNode;
  title?: React.ReactNode;
  icon?: React.ReactNode;
  badge?: React.ReactNode;
  type?: 'event' | 'run';
  metadata?: React.ReactNode;
  button?: React.ReactNode;
  active?: boolean;
}

export function ContentCard({
  children,
  title,
  icon,
  badge,
  type,
  metadata,
  button,
  active = false,
}: ContentCardProps) {
  return (
    <div
      className={cn(
        active ? `bg-canvasBase` : ``,
        `border-subtle flex flex-1 shrink-0 flex-col overflow-hidden overflow-y-auto border`
      )}
    >
      <div className={cn(title ? 'relative z-30 px-5 py-4' : '')}>
        <div className="flex items-center justify-between leading-7">
          {title ? (
            <div className="flex flex-1 items-center gap-2">
              {type === 'event' && <EventsIcon className="text-basis h-4 w-4" />}
              {type !== 'event' && icon}
              <h1 className="text-basis flex-1 text-base">{title}</h1>
            </div>
          ) : null}
          {button}
        </div>
        {badge}
        {metadata}
      </div>
      <div className="flex-1">{children}</div>
    </div>
  );
}
