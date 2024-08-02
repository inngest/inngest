import { IconEvent } from '@inngest/components/icons/Event';

import { EventsHeader } from '@/components/Events/Header';
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
  const dashboardPath = `/env/${envSlug}/events/${eventSlug}`;
  const logsPath = `/env/${envSlug}/events/${eventSlug}/logs`;
  const eventName = decodeURIComponent(eventSlug);

  return (
    <>
      {newIANav ? (
        <EventsHeader envSlug={envSlug} eventSlug={eventSlug} />
      ) : (
        <OldHeader
          icon={<IconEvent className="h-5 w-5 text-white" />}
          title={eventName}
          links={[
            {
              href: dashboardPath,
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
