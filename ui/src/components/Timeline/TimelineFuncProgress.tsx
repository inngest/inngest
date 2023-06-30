import { Time } from '../Time';

interface TimelineFuncProgressProps {
  label: string;
  date?: string | number;
  id: string;
  children?: React.ReactNode;
}

export default function TimelineFuncProgress({
  label,
  date,
  id,
  children,
}: TimelineFuncProgressProps) {
  return (
    <div className="mb-2">
      <div className="flex items-start justify-between w-full">
        <div>
          <h2 className="text-slate-50">{label}</h2>
          {date && (
            <span className="text-2xs mt-1 block leading-none text-slate-400">
              <Time date={date} />
            </span>
          )}
        </div>
        <span className="text-3xs mt-1 text-slate-500">{id}</span>
      </div>
      {children && <div className="w-full mt-4">{children}</div>}
    </div>
  );
}
