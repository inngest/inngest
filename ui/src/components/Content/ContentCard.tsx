import { IconEvent, IconFunction } from '@/icons';
import classNames from '@/utils/classnames';

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

export default function ContentCard({
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
        active ? `bg-slate-950` : ``,
        `flex-1 border rounded-lg border-slate-800/30 overflow-hidden flex flex-col shrink-0`,
      )}
    >
      <div
        className={classNames(title ? 'px-5 py-4 relative z-30' : '')}
      >
        <div className="flex items-center justify-between leading-7">
          {title ? (
            <div className="flex items-center gap-2">
              {type === 'event' && <IconEvent className="text-slate-300" />}
              {type === 'run' && <IconFunction className="text-slate-400" />}
              <h1 className="text-base text-slate-50">{title}</h1>
              {icon}
            </div>
          ) : null}
          {button}
        </div>
        {badge}
        {metadata}
      </div>
      <div className="overflow-y-auto flex-1">{children}</div>
    </div>
  );
}
