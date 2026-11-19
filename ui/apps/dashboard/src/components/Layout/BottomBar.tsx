import { Fragment } from 'react';
import { cn } from '@inngest/components/utils/classNames';

import { useSystemStatus } from '../Support/SystemStatus';
import FeedbackPopover from '../Feedback/FeedbackPopover';
import SystemStatusIcon from '../Navigation/SystemStatusIcon';

const links = [
  {
    label: 'Docs',
    href: 'https://www.inngest.com/docs?ref=app-bottom-bar',
    external: true,
  },
  { label: 'MCP', href: '/mcp/setup', external: false },
  {
    label: 'Changelog',
    href: 'https://www.inngest.com/changelog',
    external: true,
  },
  { label: 'Support', href: 'https://support.inngest.com', external: true },
];

export default function BottomBar({ focused = false }: { focused?: boolean }) {
  const status = useSystemStatus();

  return (
    <footer
      className={cn(
        'bg-canvasSubtle text-muted flex shrink-0 items-center text-xs',
        focused ? 'min-h-[30px] px-2 py-1' : 'h-[30px] px-6',
      )}
    >
      <div className={cn('flex items-center gap-3', focused && 'flex-wrap')}>
        {links.map((item, i) => (
          <Fragment key={item.label}>
            {i > 0 && <span className="text-disabled">|</span>}
            <a
              href={item.href}
              target={item.external ? '_blank' : undefined}
              rel={item.external ? 'noreferrer' : undefined}
              className="hover:text-basis"
            >
              {item.label}
            </a>
          </Fragment>
        ))}
        <FeedbackPopover
          leadingDivider
          trigger={
            <button type="button" className="hover:text-basis">
              Feedback
            </button>
          }
        />
        <span className="text-disabled">|</span>
        <a
          href="https://status.inngest.com"
          target="_blank"
          rel="noreferrer"
          className="hover:text-basis flex items-center gap-1.5"
        >
          Status
          <SystemStatusIcon status={status} className="mx-0 h-2.5 w-2.5" />
        </a>
      </div>
    </footer>
  );
}
