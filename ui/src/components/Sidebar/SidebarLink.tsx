'use client';

import Link, { type LinkProps } from 'next/link';

import classNames from '@/utils/classnames';
import { usePathname } from 'next/navigation';

interface SidebarLinkProps {
  href: string;
  icon: React.ReactNode;
  badge?: number;
}

export default function SidebarLink({ href, icon, badge = 0 }: SidebarLinkProps) {
  const pathname = usePathname();

  return (
    <Link
      href={href}
      className={classNames(
        pathname.startsWith(`/${href.split('/')[1]}`)
          ? `border-indigo-400`
          : `border-transparent opacity-40 hover:opacity-100`,
        `border-l-2 flex items-center justify-center w-full py-3 transition-all duration-150`
      )}
    >
      {icon}
    </Link>
  );
}
