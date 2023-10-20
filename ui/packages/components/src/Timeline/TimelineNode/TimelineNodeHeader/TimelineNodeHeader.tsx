import { Badge } from '@inngest/components/Badge';

type Props = {
  icon: React.ReactNode;
  badge?: string;
  title?: string;
  metadata?: {
    label: string;
    value: string;
  };
};

export function TimelineNodeHeader({ icon, badge, title, metadata }: Props) {
  return (
    <div className="flex items-start justify-between leading-7 text-slate-100	">
      <div className="mr-2 flex flex-1 items-start gap-2">
        <div className="flex items-center gap-2">
          {icon}
          {badge && (
            <Badge kind="solid" className="bg-slate-800 text-slate-400">
              {badge}
            </Badge>
          )}
        </div>
        <p className=" flex-1 text-base">{title}</p>
      </div>
      <div className="flex items-center gap-2">
        <p className="text-xs">{metadata?.label}</p>
        <p className="text-sm">{metadata?.value}</p>
      </div>
    </div>
  );
}
