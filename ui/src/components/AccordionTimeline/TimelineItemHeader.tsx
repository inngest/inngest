import Badge from '@/components/Badge';

export type TimelineItemHeaderProps = {
  icon: React.ReactNode;
  badge?: string;
  title?: string;
  metadata?: {
    label: string;
    value: string;
  };
};

export default function TimelineItemHeader({
  icon,
  badge,
  title,
  metadata,
}: TimelineItemHeaderProps) {
  return (
    <div className="text-slate-100 flex items-start justify-between leading-7	">
      <div className="flex items-start gap-2 flex-1 mr-2">
        <div className="flex items-center gap-2">
          {icon}
          {badge && (
            <Badge kind="solid" className="text-slate-400 bg-slate-800">
              {badge}
            </Badge>
          )}
        </div>
        <p className=" text-base flex-1">{title}</p>
      </div>
      <div className="flex items-center gap-2">
        <p className="text-xs">{metadata?.label}</p>
        <p className="text-sm">{metadata?.value}</p>
      </div>
    </div>
  );
}
