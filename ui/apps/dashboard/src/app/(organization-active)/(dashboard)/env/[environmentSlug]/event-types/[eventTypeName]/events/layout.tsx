import EventLogs from './EventLogs';

type EventLogsLayoutProps = {
  children: React.ReactNode;
  params: {
    eventName: string;
  };
};
export default function EventLogsLayout({ children, params }: EventLogsLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="border-muted w-80 flex-shrink-0 overflow-y-auto border-r">
        <EventLogs eventName={decodeURIComponent(params.eventName)} />
      </div>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}
