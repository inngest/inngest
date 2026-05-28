import type { ComponentType, ReactNode } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiArticleLine, RiBookReadLine, RiMailLine } from '@remixicon/react';

import { useSystemStatus } from '../Support/SystemStatus';
import SystemStatusIcon from './SystemStatusIcon';

type StripItem = {
  label: string;
  href: string;
  Icon: ComponentType<{ className?: string }>;
};

const items: StripItem[] = [
  { label: 'Support', href: 'https://support.inngest.com', Icon: RiMailLine },
  {
    label: 'Docs',
    href: 'https://www.inngest.com/docs?ref=app-sidebar',
    Icon: RiBookReadLine,
  },
  {
    label: 'Changelog',
    href: 'https://www.inngest.com/changelog',
    Icon: RiArticleLine,
  },
];

export default function UtilityStrip({ collapsed }: { collapsed: boolean }) {
  const status = useSystemStatus();

  return (
    <div
      className={cn(
        'flex items-center py-2',
        collapsed ? 'flex-col gap-1' : 'flex-row justify-between',
      )}
    >
      {items.map((item) => (
        <StripLink key={item.label} label={item.label} href={item.href}>
          <item.Icon className="h-4 w-4" />
        </StripLink>
      ))}
      <StripLink label="Status" href="https://status.inngest.com">
        <SystemStatusIcon status={status} className="mx-0 h-3.5 w-3.5" />
      </StripLink>
    </div>
  );
}

function StripLink({
  label,
  href,
  children,
}: {
  label: string;
  href: string;
  children: ReactNode;
}) {
  return (
    <OptionalTooltip tooltip={label}>
      <a
        href={href}
        target="_blank"
        rel="noreferrer"
        aria-label={label}
        className="text-muted hover:bg-canvasSubtle hover:text-basis flex h-7 w-7 items-center justify-center rounded"
      >
        {children}
      </a>
    </OptionalTooltip>
  );
}
