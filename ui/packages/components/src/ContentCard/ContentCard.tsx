import { IconEvent } from '@inngest/components/icons/Event';
import { classNames } from '@inngest/components/utils/classNames';

interface ContentCardProps {
  children: React.ReactNode;
  title?: string;
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
      className={classNames(
        active ? `bg-slate-910` : ``,
        `flex flex-1 shrink-0 flex-col overflow-hidden overflow-y-auto border border-slate-800/30`
      )}
    >
      <div className={classNames(title ? 'relative z-30 px-5 py-4' : '')}>
        <div className="flex items-center justify-between leading-7">
          {title ? (
            <div className="flex flex-1 items-center gap-2">
              {type === 'event' && <IconEvent className="text-slate-300" />}
              {type !== 'event' && icon}
              <h1 className="flex-1 text-base text-slate-50">{title}</h1>
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
