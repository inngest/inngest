import type { UrlObject } from 'url';
import type { ReactNode } from 'react';
import { OptionalLink } from '@inngest/components/Link/OptionalLink';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  prefetch = true,
}: {
  text: string;
  icon: ReactNode;
  collapsed: boolean;
  href?: string | UrlObject;
  prefetch?: boolean;
}) => {
  return (
    <OptionalLink href={href} prefetch={prefetch}>
      <div
        className={`flex cursor-pointer flex-row items-center p-2.5 ${
          collapsed ? 'justify-center ' : 'justify-start'
        }  `}
      >
        {icon}
        {!collapsed && <span className="text-muted ml-2.5 text-sm leading-tight">{text}</span>}
      </div>
    </OptionalLink>
  );
};
