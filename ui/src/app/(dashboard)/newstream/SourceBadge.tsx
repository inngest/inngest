import type { Row } from '@tanstack/react-table';

import Badge from '@/components/Badge';
import { IconClock, IconReplay, IconWebhook, IconWindow } from '@/icons';
import type { Trigger } from './page';

type SourceBadgeProps = {
  row: Row<Trigger>;
};

export default function SourceBadge({ row }: SourceBadgeProps) {
  const { type, name } = row?.original?.source;
  const { test } = row?.original;

  let icon, styles, sourceName;
  switch (type) {
    case 'replay':
      icon = <IconReplay className="h-4" />;
      styles = 'text-sky-400 bg-sky-400/10';
      break;
    case 'app':
      icon = <IconWindow className="h-4" />;
      styles = 'text-teal-400 bg-teal-400/10';
      break;
    case 'webhook':
      icon = <IconWebhook className="h-4" />;
      styles = 'text-indigo-400 bg-indigo-400/10';
      break;
    case 'scheduled':
    case 'manual':
      icon = <IconClock />;
      styles = 'text-orange-400 bg-orange-400/10 capitalize';
      sourceName = type;
      break;
    default:
      icon = null;
      styles = 'text-slate-400 bg-slate-400/10';
  }

  return (
    <span className="flex items-center gap-1">
      <Badge kind="solid" className={styles}>
        <span className="flex items-center gap-1">
          {icon}
          {name ?? sourceName}
        </span>
      </Badge>
      {test && (
        <Badge kind="solid" className="text-pink-400 bg-pink-400/10">
          Test
        </Badge>
      )}
    </span>
  );
}
