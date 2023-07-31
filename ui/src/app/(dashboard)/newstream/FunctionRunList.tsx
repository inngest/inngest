import { FunctionRunStatus } from '@/store/generated';
import {
  IconStatusCircleCheck,
  IconStatusCircleArrowPath,
  IconStatusCircleCross,
  IconStatusCircleMinus,
} from '@/icons';

const functionRunStatusIcons = {
  [FunctionRunStatus.Running]: IconStatusCircleArrowPath,
  [FunctionRunStatus.Completed]: IconStatusCircleCheck,
  [FunctionRunStatus.Failed]: IconStatusCircleCross,
  [FunctionRunStatus.Cancelled]: IconStatusCircleMinus,
} as const satisfies Record<FunctionRunStatus, React.ComponentType>;

type FunctionRunListProps = {
  functionRuns: {
    id: string;
    name: string;
    status: FunctionRunStatus;
  }[];
};

export default function FunctionRunList({
  functionRuns,
}: FunctionRunListProps) {
  return (
    <>
      {functionRuns.length < 1 ? (
        <p className="text-slate-600">No functions called</p>
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns.map((functionRun) => {
            const FunctionRunStatusIcon = functionRunStatusIcons[functionRun.status];
            return (
              <li key={functionRun.id} data-key={functionRun.id} className="flex items-center gap-2">
                <FunctionRunStatusIcon />
                {functionRun.name}
              </li>
            );
          })}
        </ul>
      )}
    </>
  );
}
