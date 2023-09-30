import { FunctionRunStatusIcons } from '@/components/Function/RunStatusIcons';
import { useGetFunctionRunStatusQuery, type FunctionRun } from '@/store/generated';

export default function FunctionRunList({ functionRuns }) {
  return (
    <>
      {!functionRuns || functionRuns.length < 1 ? (
        <p className="text-slate-600">No functions called</p>
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns &&
            functionRuns.map((functionRun) => {
              return <FunctionRunItem key={functionRun.id} functionRunID={functionRun.id} />;
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

  return (
    <li key={functionRun?.id} data-key={functionRun?.id} className="flex items-center gap-2">
      <FunctionRunStatusIcons status={functionRun.status} className="icon-xl" />
      {functionRun?.function?.name}
    </li>
  );
}
