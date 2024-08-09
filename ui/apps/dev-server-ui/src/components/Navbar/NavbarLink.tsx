import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Badge } from '@inngest/components/Badge';
import { classNames } from '@inngest/components/utils/classNames';

export type NavbarLinkProps = {
  icon: React.ReactNode;
  badge?: React.ReactNode;
  href: Route;
  hasError?: boolean;
  tabName: string;
};

export default function NavbarLink({ icon, href, tabName, hasError, badge }: NavbarLinkProps) {
  const pathname = usePathname();
  const isActive = pathname === href;

  return (
    <Link
      href={href}
      className={classNames(
        isActive
          ? `border-indigo-400 text-white`
          : `border-transparent text-slate-400 hover:text-white`,
        `flex w-full items-center justify-center gap-2 border-t-2 px-3 leading-[2.75rem] transition-all duration-150`
      )}
    >
      {icon}
      {tabName}
      {hasError && <Badge kind={'error'} />}
      {badge}
    </Link>
  );
}
