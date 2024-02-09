'use client';

import { type Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { Time } from '@/components/Time';
import ClockIcon from '@/icons/ClockIcon';
import cn from '@/utils/cn';
import { getManageKey } from '@/utils/urls';

type KeysListItemProps = {
  list: {
    id: string;
    name: string | null;
    createdAt: string;
    source: string;
  }[];
};

export default function KeysListItem({ list }: KeysListItemProps) {
  const env = useEnvironment();
  const pathname = usePathname();
  const page = getManageKey(pathname);

  // Change once there's a way to get the route param in a server component
  const filteredList = list.filter((item) => `${item.source}s` === page);

  if (filteredList.length === 0) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <h2 className="text-sm font-semibold text-gray-900">{'No ' + page + ' yet.'}</h2>
      </div>
    );
  }

  return (
    <>
      {filteredList.map((key) => {
        const eventPathname = `/env/${env.slug}/manage/${page}/${key.id}`;
        const isActive = pathname === eventPathname;

        return (
          <li key={key.id} className="border-b border-slate-100">
            <Link
              href={eventPathname as Route}
              className={cn('block px-4 py-3 hover:bg-slate-100', isActive && 'bg-slate-100')}
            >
              <p className="mb-1 text-sm font-semibold text-slate-800">{key.name}</p>
              <div className="flex items-center gap-1">
                <ClockIcon />

                <Time
                  className="truncate text-sm text-slate-700"
                  format="relative"
                  value={new Date(key.createdAt)}
                />
              </div>
            </Link>
          </li>
        );
      })}
    </>
  );
}
