import { Header } from '@inngest/components/Header/Header';
import { IconEvent } from '@inngest/components/icons/Event';

import SendEventButton from '@/components/Events/SendEventButton';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import OldHeader from '@/components/Header/old/Header';

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
  const newIANav = await getBooleanFlag('new-ia-nav');
  const eventsPath = `/env/${envSlug}/events`;
  const logsPath = `/env/${envSlug}/events/${eventSlug}/logs`;
  const eventPath = `/env/${envSlug}/events/${eventSlug}`;
  const eventName = decodeURIComponent(eventSlug);

  return (
    <>
      {newIANav ? (
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
      ) : (
        <OldHeader
          icon={<IconEvent className="h-5 w-5 text-white" />}
          title={eventName}
          links={[
            {
              href: eventPath,
              text: 'Dashboard',
              active: 'exact',
            },
            {
              href: logsPath,
              text: 'Logs',
            },
          ]}
          action={<SendEventButton eventName={eventName} />}
        />
      )}
      {children}
    </>
  );
}
