import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';

import { classNames } from '../utils/classNames';

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
        `flex flex-1 shrink-0 flex-col overflow-hidden rounded-lg border border-slate-800/30`
      )}
    >
      <div className={classNames(title ? 'relative z-30 px-5 py-4' : '')}>
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
      <div style={{ scrollbarGutter: 'stable' }} className="flex-1 overflow-hidden overflow-y-auto">
        {children}
      </div>
    </div>
  );
}
