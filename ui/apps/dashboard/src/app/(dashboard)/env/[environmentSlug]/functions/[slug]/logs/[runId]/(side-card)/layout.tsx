import TimelineIcon from '@/icons/timeline.svg';
import FunctionRunTab from './FunctionRunTab';

const tabs = [
  {
    name: 'Timeline',
    pathSegment: '(timeline)',
    icon: <TimelineIcon aria-hidden="true" />,
  },
  { name: 'Payload', pathSegment: 'payload' },
  { name: 'Output', pathSegment: 'output' },
] as const;

type FunctionRunDetailsCardLayoutProps = {
  params: {
    environmentSlug: string;
    slug: string;
    runId: string;
  };
  children: React.ReactNode;
};

export default async function FunctionRunDetailsCardLayout({
  params,
  children,
}: FunctionRunDetailsCardLayoutProps) {
  return (
    <div className="flex h-full flex-col space-y-1.5 rounded-xl bg-slate-900 text-white">
      <nav className="bg-slate-910 flex gap-2 rounded-t-xl px-4" aria-label="Tabs">
        {tabs.map((tab) => (
          <FunctionRunTab
            icon={tab.name === 'Timeline' ? tab.icon : undefined}
            basePathname={`/env/${params.environmentSlug}/functions/${params.slug}/logs/${params.runId}/`}
            pathSegment={tab.pathSegment}
            key={tab.name}
          >
            {tab.name}
          </FunctionRunTab>
        ))}
      </nav>
      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">{children}</div>
    </div>
  );
}
