import { ChartBarSquareIcon, CommandLineIcon } from '@heroicons/react/20/solid';

import Header, { type HeaderLink } from '@/components/Header/Header';
import EventIcon from '@/icons/event.svg';
import SendEventButton from './SendEventButton';

type EventLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    eventName: string;
  };
};

export default function EventLayout({ children, params }: EventLayoutProps) {
  const navLinks: HeaderLink[] = [
    {
      href: `/env/${params.environmentSlug}/events/${params.eventName}`,
      text: 'Dashboard',
      icon: <ChartBarSquareIcon className="w-3.5" />,
      active: 'exact',
    },
    {
      href: `/env/${params.environmentSlug}/events/${params.eventName}/logs`,
      text: 'Logs',
      icon: <CommandLineIcon className="w-3.5" />,
    },
  ];

  return (
    <>
      <Header
        icon={<EventIcon className="h-5 w-5 text-white" />}
        title={decodeURIComponent(params.eventName)}
        links={navLinks}
        action={
          <SendEventButton
            environmentSlug={params.environmentSlug}
            eventName={decodeURIComponent(params.eventName)}
          />
        }
      />
      {children}
    </>
  );
}
