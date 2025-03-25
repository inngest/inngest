'use client';

import { type Route } from 'next';
import NextLink from 'next/link';
import { usePathname } from 'next/navigation';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';
import { RiTimeLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { getManageKey } from '@/utils/urls';

type KeysListItemProps = {
  list: {
    id: string;
    name: string | null;
    createdAt: string;
    source: string;
  }[];
};

const pageFilters: { [key: string]: string[] } = {
  keys: ['key', 'integration'],
  webhooks: ['webhook'],
};

export default function KeysListItem({ list }: KeysListItemProps) {
  const env = useEnvironment();
  const pathname = usePathname();
  const page = getManageKey(pathname);

  // Change once there's a way to get the route param in a server component
  const filteredList = page ? list.filter((item) => pageFilters[page]?.includes(item.source)) : [];

  if (filteredList.length === 0) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <h2 className="text-basis text-sm font-semibold">{'No ' + page + ' yet.'}</h2>
      </div>
    );
  }

  return (
    <>
      {filteredList.map((key) => {
        const eventPathname = `/env/${env.slug}/manage/${page}/${key.id}`;
        const isActive = pathname === eventPathname;

        return (
          <li key={key.id} className="border-subtle text-basis border-b">
            <NextLink
              href={eventPathname as Route}
              className={cn('hover:bg-canvasMuted block px-4 py-3', isActive && 'bg-canvasSubtle')}
            >
              <p className="mb-1 text-sm font-semibold">{key.name}</p>
              <div className="flex items-center gap-1">
                <RiTimeLine className="h-4 w-4" />

                <Time
                  className="truncate text-sm"
                  format="relative"
                  value={new Date(key.createdAt)}
                />
              </div>
            </NextLink>
          </li>
        );
      })}
    </>
  );
}
