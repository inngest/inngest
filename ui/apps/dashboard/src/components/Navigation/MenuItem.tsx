import type { ReactNode } from 'react';
import Link from 'next/link';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
}: {
  text: string;
  icon: ReactNode;
  collapsed: boolean;
  href: string;
}) => {
  return (
    <Link href={href}>
      <div
        className={`flex cursor-pointer flex-row items-center p-2.5 ${
          collapsed ? 'justify-center ' : 'justify-start'
        }  `}
      >
        {icon}
        {!collapsed && <span className="text-muted ml-2.5 text-sm leading-tight">{text}</span>}
      </div>
    </Link>
  );
};
