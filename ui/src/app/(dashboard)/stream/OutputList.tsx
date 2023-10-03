import renderRunOutput from '@/components/Function/RunOutputRenderer';
import { useGetFunctionRunOutputQuery, type FunctionRun } from '@/store/generated';

export default function OutputList({ functionRuns }) {
  return (
    <>
      {!functionRuns || functionRuns.length < 1 ? (
        <p className="text-slate-600" />
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns &&
            functionRuns.map((functionRun) => {
              return <OutputItem key={functionRun.id} functionRunID={functionRun.id} />;
            })}
        </ul>
      )}
    </>
  );
}

type FunctionRunStatusSubset = Pick<FunctionRun, 'id' | 'status' | 'output'>;

export function OutputItem({ functionRunID }) {
  const { data } = useGetFunctionRunOutputQuery({ id: functionRunID }, { pollingInterval: 1500 });
  const functionRun = (data?.functionRun as FunctionRunStatusSubset) || {};

  if (!functionRun || !functionRun?.output) {
    return null;
  }

  const { message, errorName, output } = renderRunOutput(functionRun);

  return (
    <li
      key={functionRun?.id}
      data-key={functionRun?.id}
      className="flex items-baseline gap-2 font-mono"
    >
      {errorName && <span className={'font-bold text-rose-500'}>{errorName}</span>}
      <span className="text-xs">{message}</span>
      {(!errorName || !message) && <span className="text-xs">{output}</span>}
    </li>
  );
}
