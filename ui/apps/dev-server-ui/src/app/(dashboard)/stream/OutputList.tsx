import { renderOutput } from '@inngest/components/utils/outputRenderer';

import {
  FunctionRunStatus,
  useGetFunctionRunOutputQuery,
  type FunctionRun,
} from '@/store/generated';

type OutputListProps = {
  functionRuns: FunctionRun[];
};

export default function OutputList({ functionRuns }: OutputListProps) {
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

export function OutputItem({ functionRunID }: { functionRunID: string }) {
  const { data } = useGetFunctionRunOutputQuery({ id: functionRunID }, { pollingInterval: 1500 });
  const functionRun = (data?.functionRun as FunctionRunStatusSubset) || {};

  if (!functionRun || !functionRun?.output || !functionRun?.status) {
    return null;
  }

  const { message, errorName, output } = renderOutput({
    isSuccess: functionRun.status === FunctionRunStatus.Completed,
    content: functionRun.output,
  });

  return (
    <li
      key={functionRun?.id}
      data-key={functionRun?.id}
      className="flex items-baseline gap-2 font-mono"
    >
      {errorName && <span className={'font-bold text-rose-500'}>{errorName}</span>}
      {message && <span className="text-xs">{message}</span>}
      {(!errorName || !message) && <span className="text-xs">{output}</span>}
    </li>
  );
}
