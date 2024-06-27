import { RunStatusIcon } from '@inngest/components/FunctionRunStatusIcons';
import { cn } from '@inngest/components/utils/classNames';

import type { FunctionRunStatus } from '../types/functionRun';

interface FuncCardProps {
  title: string;
  id: string;
  status?: FunctionRunStatus;
  active?: boolean;
  onClick?: () => void;
}

export function FuncCard({ title, id, status, active = false, onClick }: FuncCardProps) {
  return (
    <a
      className={cn(
        active
          ? `outline-offset-3 border-slate-700/50 bg-slate-900 outline outline-2 outline-indigo-400`
          : undefined,
        `block w-full overflow-hidden rounded-lg bg-slate-800/50`,
        onClick ? 'cursor-pointer hover:bg-slate-800/80' : undefined
      )}
      onClick={
        onClick
          ? (e) => {
              e.preventDefault();
              onClick();
            }
          : undefined
      }
    >
      <div className="flex items-center gap-2 px-5 py-3.5">
        {status && <RunStatusIcon status={status} className="h-4 w-4" />}
        <h2 className="text-white">{title}</h2>
      </div>
      <hr className="border-slate-800/50" />
      <div className="px-5 py-3.5 text-xs leading-none text-slate-500">
        Run ID: <span className="font-mono">{id}</span>
      </div>
    </a>
  );
}
