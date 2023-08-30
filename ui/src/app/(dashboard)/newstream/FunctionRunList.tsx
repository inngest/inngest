import {
  IconStatusCircleArrowPath,
  IconStatusCircleCheck,
  IconStatusCircleCross,
  IconStatusCircleMinus,
} from '@/icons';
import {
  FunctionRunStatus,
  useGetFunctionRunStatusQuery,
  type FunctionRun,
} from '@/store/generated';

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
              return <FunctionRunItem functionRunID={functionRun.id} />;
            })}
        </ul>
      )}
    </>
  );
}

type FunctionRunStatusSubset = Pick<FunctionRun, 'id' | 'function' | 'status'>;

export function FunctionRunItem({ functionRunID }) {
  const { data } = useGetFunctionRunStatusQuery({ id: functionRunID }, { pollingInterval: 1500 });
  const functionRun = (data?.functionRun as FunctionRunStatusSubset) || {};

  if (!functionRun || !functionRun?.function?.name || !functionRun.status) {
    return null;
  }
  const FunctionRunStatusIcon = functionRunStatusIcons[functionRun.status];

  return (
    <li key={functionRun?.id} data-key={functionRun?.id} className="flex items-center gap-2">
      <FunctionRunStatusIcon />
      {functionRun?.function?.name}
    </li>
  );
}
