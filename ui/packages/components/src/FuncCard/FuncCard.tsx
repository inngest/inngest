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
        active ? `border-muted bg-canvasBase border` : undefined,
        `bg-canvasSubtle block w-full overflow-hidden rounded-md`,
        onClick ? 'hover:bg-surfaceMuted cursor-pointer' : undefined
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
        <h2 className="text-basis">{title}</h2>
      </div>
      <hr className="border-muted" />
      <div className="text-subtle px-5 py-3.5 text-xs leading-none">
        Run ID: <span className="font-mono">{id}</span>
      </div>
    </a>
  );
}
