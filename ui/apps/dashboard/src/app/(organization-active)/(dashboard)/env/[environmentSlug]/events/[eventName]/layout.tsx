import { Header } from '@inngest/components/Header/Header';

import SendEventButton from '@/components/Events/SendEventButton';

type EventLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    eventName: string;
  };
};

export default async function EventLayout({
  children,
  params: { environmentSlug: envSlug, eventName: eventSlug },
}: EventLayoutProps) {
  const eventsPath = `/env/${envSlug}/events`;
  const logsPath = `/env/${envSlug}/events/${eventSlug}/logs`;
  const eventPath = `/env/${envSlug}/events/${eventSlug}`;
  const eventName = decodeURIComponent(eventSlug);

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Events', href: eventsPath },
          { text: eventName, href: eventPath },
        ]}
        tabs={[
          {
            href: eventPath,
            children: 'Dashboard',
            exactRouteMatch: true,
          },
          {
            href: logsPath,
            children: 'Logs',
          },
        ]}
        action={<SendEventButton eventName={eventName} newIANav={true} />}
      />

      {children}
    </>
  );
}
