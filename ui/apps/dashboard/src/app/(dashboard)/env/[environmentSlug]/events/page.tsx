'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ArrowRightIcon, ChartBarIcon, CodeBracketSquareIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import SendEventButton from '@/app/(dashboard)/env/[environmentSlug]/events/[eventName]/SendEventButton';
import MiniStackedBarChart from '@/components/Charts/MiniStackedBarChart';
import Header from '@/components/Header/Header';
import HorizontalPillList from '@/components/Pill/HorizontalPillList';
import { Pill } from '@/components/Pill/Pill';
import LoadingIcon from '@/icons/LoadingIcon';
import EventIcon from '@/icons/event.svg';
import { useEventTypes } from '@/queries';
import cn from '@/utils/cn';
import EventListNotFound from './EventListNotFound';

type EventTypesPageProps = {
  params: {
    environmentSlug: string;
  };
};

export const runtime = 'nodejs';

export default function EventTypesPage({ params }: EventTypesPageProps) {
  const [pages, setPages] = useState([1]);

  function appendPage() {
    setPages((prevPages) => {
      const lastPage = prevPages[prevPages.length - 1] ?? 0;
      return [...prevPages, lastPage + 1];
    });
  }

  return (
    <>
      <Header
        title="Events"
        icon={<EventIcon className="h-4 w-4 text-white" />}
        action={<SendEventButton environmentSlug={params.environmentSlug} />}
      />

      <main className="min-h-0 flex-1 overflow-y-auto bg-slate-100">
        <table className="relative w-full border-b border-slate-200 bg-white">
          <thead className="shadow-outline-primary-light sticky top-0 z-10 bg-white">
            <tr>
              {['Event Name', 'Functions', 'Volume (24hr)'].map((heading, index) => (
                <th
                  key={heading}
                  scope="col"
                  className={cn(
                    'whitespace-nowrap px-2 py-3 text-left text-xs font-semibold text-slate-600',
                    index === 0 && 'pl-6'
                  )}
                >
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="h-full divide-y divide-slate-100 px-4 py-3">
            {pages.map((page) => (
              <EventTypesListPaginationPage
                key={page}
                environmentSlug={params.environmentSlug}
                isLastLoadedPage={page === pages[pages.length - 1]}
                page={page}
                onLoadMore={appendPage}
              />
            ))}
          </tbody>
        </table>
      </main>
    </>
  );
}

type EventListPaginationPageProps = {
  environmentSlug: string;
  isLastLoadedPage: boolean;
  page: number;
  onLoadMore: () => void;
};

function EventTypesListPaginationPage({
  environmentSlug,
  isLastLoadedPage,
  page,
  onLoadMore,
}: EventListPaginationPageProps) {
  const [{ data, fetching: isFetchingEvents }] = useEventTypes({
    environmentSlug,
    page,
  });

  const events = data?.workspace.events.data ?? [];
  const totalPages = data?.workspace.events.page.totalPages ?? 1;
  const hasNextPage = page < totalPages;
  const isFirstPage = page === 1;

  if (isFetchingEvents) {
    return (
      <tr>
        <td colSpan={3} className="h-56">
          <div className="relative flex items-center justify-center">
            <LoadingIcon />
          </div>
        </td>
      </tr>
    );
  }

  if (isFirstPage && events.length === 0) {
    return (
      <tr>
        <td colSpan={3}>
          <EventListNotFound environmentSlug={environmentSlug} />
        </td>
      </tr>
    );
  }

  return (
    <>
      {events.map((event) => {
        const dailyVolume = event.dailyVolume.total;

        // Creates an array of objects containing the volume for each usage slot (1 hour)
        const dailyVolumeSlots = event.dailyVolume.data.map((volumeSlot) => ({
          startCount: volumeSlot.count,
        }));

        return (
          <tr className="hover:bg-slate-50" key={event.name}>
            <td className="w-96 whitespace-nowrap">
              <div className="flex items-center gap-2.5 pl-4">
                <Link
                  href={`/env/${environmentSlug}/events/${encodeURIComponent(event.name)}`}
                  className="group flex w-full items-center gap-2 px-2 py-3 text-sm font-medium text-slate-700  hover:text-indigo-600"
                >
                  {event.name}
                  <ArrowRightIcon className="h-3 w-3 -translate-x-3 text-indigo-600 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                </Link>
              </div>
            </td>
            <td className="space-x-2 whitespace-nowrap px-2 py-3">
              <HorizontalPillList
                alwaysVisibleCount={2}
                pills={event.functions.map((function_) => (
                  <Pill
                    href={`/env/${environmentSlug}/functions/${encodeURIComponent(function_.slug)}`}
                    key={function_.name}
                    className="bg-white align-middle text-slate-600"
                  >
                    <CodeBracketSquareIcon className="mr-1 h-3.5 w-3.5 text-indigo-500" />
                    {function_.name}
                  </Pill>
                ))}
              />
            </td>
            <td className="w-60 py-1 pl-2 pr-6">
              <div className="flex w-56 items-center justify-end gap-2">
                <Pill className="gap-1 bg-white align-middle text-slate-600">
                  <ChartBarIcon className="-ml-0.5 h-3.5 w-3.5 shrink-0 text-indigo-500" />
                  <span className="overflow-hidden whitespace-nowrap text-xs text-slate-600">
                    {dailyVolume.toLocaleString(undefined, {
                      notation: 'compact',
                      compactDisplay: 'short',
                    })}
                  </span>
                </Pill>
                <MiniStackedBarChart className="shrink-0" data={dailyVolumeSlots} />
              </div>
            </td>
          </tr>
        );
      })}

      {isLastLoadedPage && hasNextPage && (
        <tr>
          <td colSpan={3} className="py-2.5 text-center">
            <Button appearance="outlined" btnAction={onLoadMore} label="Load More" />
          </td>
        </tr>
      )}
    </>
  );
}
