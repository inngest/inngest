import { Fragment } from 'react';

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

export default function BottomBar() {
  const status = useSystemStatus();

  return (
    <footer className="bg-canvasSubtle text-muted flex h-[30px] shrink-0 items-center px-6 text-xs">
      <div className="flex items-center gap-3">
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
