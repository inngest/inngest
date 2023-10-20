import { CommandLineIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

type FunctionCardTypes = {
  name: string;
  trigger: string;
  time: string;
  version: string;
  id: string;
  active: boolean;
  status: 'success' | 'error' | 'sleep' | 'running' | 'scheduled' | 'paused' | 'waiting';
};

export default function FunctionCard({
  name,
  trigger,
  time,
  version,
  id,
  status,
  active = false,
}: FunctionCardTypes) {
  return (
    <div className="overflow-hidden rounded-lg border border-slate-200 bg-slate-50 shadow-sm">
      <div className="flex flex-col gap-1 bg-white px-5 py-4">
        <div className="flex items-center justify-between">
          <h4 className="flex items-center gap-1 text-sm font-semibold text-slate-700">
            <CommandLineIcon className="w-4 text-slate-600" />
            {name}
          </h4>
          <span className="text-xs text-slate-500">{version}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-slate-500">{time}</span>
          <span className="text-xs text-slate-500">{id}</span>
        </div>
      </div>
      <div className="flex items-center justify-between px-4 py-2">
        <span>Status</span>
        <Button size="small" label="Pause" />
      </div>
    </div>
  );
}
