import {
  IconStatusCircleArrowPath,
  IconStatusCircleCheck,
  IconStatusCircleCross,
  IconStatusCircleMinus,
} from '@/icons';
import { FunctionRunStatus } from '@/store/generated';

const functionRunStatusIcons = {
  [FunctionRunStatus.Running]: IconStatusCircleArrowPath,
  [FunctionRunStatus.Completed]: IconStatusCircleCheck,
  [FunctionRunStatus.Failed]: IconStatusCircleCross,
  [FunctionRunStatus.Cancelled]: IconStatusCircleMinus,
} as const satisfies Record<FunctionRunStatus, React.ComponentType>;

export default function FunctionRunList({ functionRuns }) {
  return (
    <>
      {!functionRuns || functionRuns.length < 1 ? (
        <p className="text-slate-600">No functions called</p>
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns &&
            functionRuns.map((functionRun) => {
              if (!functionRun || !functionRun.function || !functionRun.status) {
                return null;
              }
              const FunctionRunStatusIcon = functionRunStatusIcons[functionRun.status];
              return (
                <li
                  key={functionRun.functionID}
                  data-key={functionRun.functionID}
                  className="flex items-center gap-2"
                >
                  <FunctionRunStatusIcon />
                  {functionRun.function.name}
                </li>
              );
            })}
        </ul>
      )}
    </>
  );
}
