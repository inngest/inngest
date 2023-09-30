import { useGetFunctionRunOutputQuery, type FunctionRun } from '@/store/generated';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';

export default function OutputList({ functionRuns }) {
  return (
    <>
      {!functionRuns || functionRuns.length < 1 ? (
        <p className="text-slate-600" />
      ) : (
        <ul className="flex flex-col space-y-4">
          {functionRuns &&
            functionRuns.map((functionRun) => {
              return <OutputItem functionRunID={functionRun.id} />;
            })}
        </ul>
      )}
    </>
  );
}

type FunctionRunStatusSubset = Pick<FunctionRun, 'id' | 'output'>;

export function OutputItem({ functionRunID }) {
  const { data } = useGetFunctionRunOutputQuery({ id: functionRunID }, { pollingInterval: 1500 });
  const functionRun = (data?.functionRun as FunctionRunStatusSubset) || {};

  if (!functionRun || !functionRun?.output) {
    return null;
  }

  let message = '';
  let errorName = '';
  const isOutputTooLarge = functionRun.output?.length > maxRenderedOutputSizeBytes;
  if (functionRun.output && !isOutputTooLarge) {
    const parsedOutput = JSON.parse(functionRun.output);

    if (parsedOutput.body && typeof parsedOutput.body === 'object') {
      message = parsedOutput.body?.message;
      errorName = parsedOutput.body?.name;
    } else if (parsedOutput.body && typeof parsedOutput.body === 'string') {
      const parsedBody = JSON.parse(parsedOutput.body);
      message = parsedBody?.message;
      errorName = parsedBody?.name;
    }
  }

  return (
    <li
      key={functionRun?.id}
      data-key={functionRun?.id}
      className="flex items-center gap-2 font-mono"
    >
      {errorName && <span className={'font-bold text-rose-500'}>{errorName}</span>}
      <span className="text-xs truncate">{message}</span>
    </li>
  );
}
