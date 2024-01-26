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
        icon={<EventIcon className="h-5 w-5 text-white" />}
        title={decodeURIComponent(params.eventName)}
        links={navLinks}
        action={<SendEventButton eventName={decodeURIComponent(params.eventName)} />}
      />
      {children}
    </>
  );
}
