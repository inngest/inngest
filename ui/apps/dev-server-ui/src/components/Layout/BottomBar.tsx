import { Fragment } from 'react';

import { useInfoQuery } from '@/store/devApi';

const links = [
  { label: 'Support', href: 'https://support.inngest.com' },
  {
    label: 'Docs',
    href: 'https://www.inngest.com/docs?ref=dev-server-bottom-bar',
  },
  { label: 'Changelog', href: 'https://www.inngest.com/changelog' },
];

export default function BottomBar() {
  const { data: info } = useInfoQuery();

  return (
    <footer className="bg-canvasSubtle text-muted flex h-[30px] shrink-0 items-center justify-between px-6 text-xs">
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
      </div>
      {/* Version is best-effort — skip the label entirely when the server
          doesn't report one. */}
      {info?.version && <div>Dev Server v{info.version}</div>}
    </footer>
  );
}
