import statusStyles from '@/utils/statusStyles';
import { FunctionRunStatus } from '@/store/generated';

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
            const itemStatus = statusStyles(functionRun.status);
            return (
              <div key={functionRun.id} className="flex items-center gap-2">
                <itemStatus.icon />
                <span>{functionRun.name}</span>
              </div>
            );
          })}
        </ul>
      )}
    </>
  );
}
