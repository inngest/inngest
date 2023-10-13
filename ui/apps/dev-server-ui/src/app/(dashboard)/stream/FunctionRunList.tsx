import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';

import { useGetFunctionRunStatusQuery, type FunctionRun } from '@/store/generated';

type FunctionRunList = {
  functionRuns: FunctionRun[];
};

export default function FunctionRunList({ functionRuns }: FunctionRunList) {
  return (
    <>
      {!functionRuns || functionRuns.length < 1 ? (
        <p className="text-slate-600">No functions called</p>
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns &&
            functionRuns
              ?.slice()
              .sort((a, b) => (a.function?.name || '').localeCompare(b.function?.name || ''))
              .map((functionRun) => {
                return <FunctionRunItem key={functionRun.id} functionRunID={functionRun.id} />;
              })}
        </ul>
      )}
    </>
  );
}

type FunctionRunStatusSubset = Pick<FunctionRun, 'id' | 'function' | 'status'>;

export function FunctionRunItem({ functionRunID }: { functionRunID: string }) {
  const { data } = useGetFunctionRunStatusQuery({ id: functionRunID }, { pollingInterval: 1500 });
  const functionRun = (data?.functionRun as FunctionRunStatusSubset) || {};

  if (!functionRun || !functionRun?.function?.name || !functionRun.status) {
    return null;
  }

  return (
    <li key={functionRun?.id} data-key={functionRun?.id} className="flex items-center gap-2">
      <FunctionRunStatusIcon status={functionRun.status} className="h-5 w-5" />
      {functionRun?.function?.name}
    </li>
  );
}
