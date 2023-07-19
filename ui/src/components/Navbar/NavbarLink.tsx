import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import classNames from '@/utils/classnames';
import Badge from '@/components/Badge';

interface NavbarLinkProps {
  icon: React.ReactNode;
  href: Route;
  badge?: number;
  hasError?: boolean;
  tabName: string;
}

export default function NavbarLink({
  icon,
  href,
  badge,
  tabName,
  hasError,
}: NavbarLinkProps) {
  const pathname = usePathname();
  const isActive = pathname === '/' + href;

  return (
    <Link
      href={href}
      className={classNames(
        isActive
          ? `border-indigo-400 text-white`
          : `border-transparent text-slate-400 hover:text-white`,
        `border-t-2 flex items-center justify-center w-full px-3 leading-[2.75rem] transition-all duration-150 gap-2`
      )}
    >
      {icon}
      {tabName}
      {typeof badge === 'number' && (
        <Badge kind={hasError ? 'error' : 'outlined'}>{badge.toString()}</Badge>
      )}
    </Link>
  );
}
