import { BatchSize } from '@inngest/components/BatchSize';
import { RunStatusIcon } from '@inngest/components/FunctionRunStatusIcons';

import { useGetFunctionRunStatusQuery, type FunctionRun } from '@/store/generated';

type FunctionRunList = {
  inBatch: boolean;
  functionRuns: FunctionRun[];
};

export default function FunctionRunList({ inBatch, functionRuns }: FunctionRunList) {
  return (
    <>
      {!functionRuns || functionRuns.length < 1 ? (
        <p className="text-subtle">{inBatch ? 'Added to batch' : 'No functions called'}</p>
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns &&
            functionRuns
              ?.slice()
              .sort((a, b) => {
                // Append with run ID to ensure unique keys. Rerunning
                // intentionally results in duplicate function names
                const aVal = `${a.function?.name || ''}${a.id}`;
                const bVal = `${b.function?.name || ''}${b.id}`;

                return aVal.localeCompare(bVal);
              })
              .map((functionRun) => {
                let batchSize;
                if (functionRun.batchID) {
                  batchSize = functionRun.events?.length;
                }

                return (
                  <FunctionRunItem
                    batchSize={batchSize}
                    key={functionRun.id}
                    functionRunID={functionRun.id}
                  />
                );
              })}
        </ul>
      )}
    </>
  );
}

type FunctionRunStatusSubset = Pick<FunctionRun, 'id' | 'function' | 'status'>;

export function FunctionRunItem({
  batchSize,
  functionRunID,
}: {
  batchSize: number | undefined;
  functionRunID: string;
}) {
  const { data } = useGetFunctionRunStatusQuery({ id: functionRunID }, { pollingInterval: 1500 });
  const functionRun = (data?.functionRun as FunctionRunStatusSubset) || {};

  if (!functionRun || !functionRun?.function?.name || !functionRun.status) {
    return null;
  }

  return (
    <li key={functionRun?.id} data-key={functionRun?.id} className="flex items-center gap-2">
      <RunStatusIcon status={functionRun.status} className="h-5 w-5" />
      {functionRun?.function?.name}
      {batchSize && <BatchSize eventCount={batchSize} />}
    </li>
  );
}
