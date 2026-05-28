import { Fragment } from 'react';
import { RiLinksLine } from '@remixicon/react';

import { useSystemStatus } from '../Support/SystemStatus';
import SystemStatusIcon from '../Navigation/SystemStatusIcon';

const links = [
  { label: 'Support', href: 'https://support.inngest.com' },
  { label: 'Docs', href: 'https://www.inngest.com/docs?ref=app-bottom-bar' },
  { label: 'Changelog', href: 'https://www.inngest.com/changelog' },
];

export default function BottomBar() {
  const status = useSystemStatus();

  return (
    <footer className="bg-canvasSubtle text-muted flex shrink-0 items-center justify-between px-4 py-2 text-sm">
      <div className="flex items-center gap-3">
        {links.map((item, i) => (
          <Fragment key={item.label}>
            {i > 0 && <span className="text-disabled">|</span>}
            <a
              href={item.href}
              target="_blank"
              rel="noreferrer"
              className="hover:text-basis"
            >
              {item.label}
            </a>
          </Fragment>
        ))}
        <span className="text-disabled">|</span>
        <a
          href="https://status.inngest.com"
          target="_blank"
          rel="noreferrer"
          className="hover:text-basis flex items-center gap-1.5"
        >
          Status
          <SystemStatusIcon status={status} className="mx-0 h-3 w-3" />
        </a>
      </div>
      <button
        type="button"
        className="bg-canvasBase border-muted text-muted hover:text-basis hover:bg-canvasSubtle/60 flex items-center gap-1.5 rounded border px-2.5 py-1"
      >
        <RiLinksLine className="h-3.5 w-3.5" />
        Ask Inngest
      </button>
    </footer>
  );
}
