'use client';

import { IconEvent } from '@inngest/components/icons/Event';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import Header, { type HeaderLink } from '@/components/Header/Header';
import SendEventButton from './SendEventButton';

type EventLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    eventName: string;
  };
};

export default function EventLayout({ children, params }: EventLayoutProps) {
  const env = useEnvironment();

  const navLinks: HeaderLink[] = [
    {
      href: `/env/${params.environmentSlug}/events/${params.eventName}`,
      text: 'Dashboard',
      active: 'exact',
    },
    {
      href: `/env/${params.environmentSlug}/events/${params.eventName}/logs`,
      text: 'Logs',
    },
  ];

  return (
    <>
      <Header
        icon={<IconEvent className="h-5 w-5 text-white" />}
        title={decodeURIComponent(params.eventName)}
        links={navLinks}
        action={
          !env.isArchived && <SendEventButton eventName={decodeURIComponent(params.eventName)} />
        }
      />
      {children}
    </>
  );
}
