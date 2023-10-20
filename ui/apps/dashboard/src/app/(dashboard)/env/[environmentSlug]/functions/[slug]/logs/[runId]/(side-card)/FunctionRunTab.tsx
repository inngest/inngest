'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { useSelectedLayoutSegment } from 'next/navigation';

import cn from '@/utils/cn';

/**
 * Check if a path segment is a route group. Route groups don't affect the URL structure.
 *
 * @see {@link https://beta.nextjs.org/docs/routing/defining-routes#route-groups}
 */
function isRouteGroup(pathSegment: string): boolean {
  const regex = /^\(.*\)$/;
  return regex.test(pathSegment);
}

type FunctionRunSideCardTabProps<PassedPathname extends string> = {
  basePathname: Route<PassedPathname>;
  pathSegment: string;
  children: React.ReactNode;
  icon?: React.ReactNode;
};
export default function FunctionRunTab<PassedPathname extends string>({
  basePathname,
  pathSegment,
  icon,
  children,
}: FunctionRunSideCardTabProps<PassedPathname>) {
  const selectedSegment = useSelectedLayoutSegment();

  const isActive = pathSegment === selectedSegment;
  const href = isRouteGroup(pathSegment) ? basePathname : basePathname + pathSegment;

  return (
    <Link
      href={href as Route}
      className={cn(
        isActive
          ? 'border-indigo-400 text-white'
          : 'border-transparent text-slate-400 hover:border-slate-400 hover:text-white',
        'group inline-flex items-center whitespace-nowrap border-b-2 p-4 text-sm font-medium'
      )}
      aria-current={isActive ? 'page' : undefined}
    >
      {icon && (
        <div
          className={cn(
            isActive ? 'text-indigo-400' : 'text-slate-400 group-hover:text-slate-600',
            '-ml-1.5 mr-1.5 h-4 w-4'
          )}
        >
          {icon}
        </div>
      )}
      {children}
    </Link>
  );
}
