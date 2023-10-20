import { Badge } from '@inngest/components/Badge';
import { IconClock } from '@inngest/components/icons/Clock';
import type { Row } from '@tanstack/react-table';

import { IconReplay, IconWebhook, IconWindow } from '@/icons';

type Item = {
  source: {
    type: string;
    name: string;
  };
  test: boolean;
};

type SourceBadgeProps = {
  row: Row<Item>;
};

export default function SourceBadge({ row }: SourceBadgeProps) {
  const { type, name } = row?.original?.source;
  const { test } = row?.original;

  let icon, styles, sourceName;
  switch (type) {
    case 'replay':
      icon = <IconReplay />;
      styles = 'text-sky-400 bg-sky-400/10';
      break;
    case 'app':
      icon = <IconWindow />;
      styles = 'text-teal-400 bg-teal-400/10';
      break;
    case 'webhook':
      icon = <IconWebhook />;
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
        <Badge kind="solid" className="bg-pink-400/10 text-pink-400">
          Test
        </Badge>
      )}
    </span>
  );
}
