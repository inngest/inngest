'use client';

import { usePathname } from 'next/navigation';

import { Header } from '@/components/Header/Header';
import SendEventButton from './SendEventButton';

type EventType = {
  envSlug: string;
  eventSlug: string;
};

//
// We must wrap our header in use client to do dynamic breadcrumbs because next does
// not want you to detect paths in layouts.
// https://github.com/vercel/next.js/issues/57762#issuecomment-1802052863
export const EventsHeader = ({ envSlug, eventSlug }: EventType) => {
  const dashboardPath = `/env/${envSlug}/events/${eventSlug}`;
  const eventPath = `/env/${envSlug}/events/${eventSlug}`;
  const logsPath = `/env/${envSlug}/events/${eventSlug}/logs`;
  const eventName = decodeURIComponent(eventSlug);
  const pathname = usePathname();

  return (
    <Header
      breadcrumb={[
        { text: 'Dashboard', href: dashboardPath },
        { text: eventName, href: eventPath },
        ...(pathname?.includes('/logs') ? [{ text: 'Logs', href: logsPath }] : []),
      ]}
      tabs={[
        {
          href: dashboardPath,
          children: 'Dashboard',
          exactRouteMatch: true,
        },
        {
          href: logsPath,
          children: 'Logs',
        },
      ]}
      action={<SendEventButton eventName={eventName} />}
    />
  );
};
