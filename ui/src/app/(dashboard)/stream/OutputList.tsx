  import {
    useGetFunctionRunOutputQuery,
    type FunctionRun,
  } from '@/store/generated';
  
  export default function OutputList({ functionRuns }) {
    return (
      <>
        {!functionRuns || functionRuns.length < 1 ? (
          <p className="text-slate-600" />
        ) : (

          <ul className="">
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
    const parsedOutput = JSON.parse(functionRun.output)
    const wasSuccessful = parsedOutput.body?.success || (parsedOutput.status.toString().startsWith('2'));
  
    return (
      <li key={functionRun?.id} data-key={functionRun?.id} className="flex gap-2 font-mono">
        <span className={wasSuccessful ? 'text-teal-300' : 'text-rose-500'}>{parsedOutput.status}</span>
        <span className='text-xs'>{parsedOutput.body?.message}</span>
      </li>
    );
  }
  