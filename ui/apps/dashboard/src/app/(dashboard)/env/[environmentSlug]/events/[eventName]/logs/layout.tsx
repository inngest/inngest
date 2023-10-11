import EventLogs from './EventLogs';

type EventLogsLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    eventName: string;
  };
};
export default function EventLogsLayout({ children, params }: EventLogsLayoutProps) {
  return (
    <div className="flex min-h-0 flex-1">
      <div className="w-80 flex-shrink-0 overflow-y-auto border-r border-slate-300">
        <EventLogs
          environmentSlug={params.environmentSlug}
          eventName={decodeURIComponent(params.eventName)}
        />
      </div>
      <div className="min-w-0 flex-1 bg-slate-50">{children}</div>
    </div>
  );
}
