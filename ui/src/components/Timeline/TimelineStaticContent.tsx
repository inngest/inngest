import { ComponentChild } from "preact";

interface TimelineStaticRowProps {
  label: string;
  date?: Date;
  actionBtn?: ComponentChild;
}

export default function TimelineStaticRow({
  label,
  date,
  actionBtn,
}: TimelineStaticRowProps) {
  return (
    <div className="flex items-start justify-between w-full pt-[2px]">
      <div>
        <h2 className="text-slate-50">{label}</h2>
        {date && (
          <span className="text-2xs mt-1 block leading-none text-slate-400">
            {date.toISOString()}
          </span>
        )}
      </div>
      {actionBtn && actionBtn}
    </div>
  );
}
