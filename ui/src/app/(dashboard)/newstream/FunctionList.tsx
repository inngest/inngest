import statusStyles from '@/utils/statusStyles';

type FunctionListProps = {
  functions: {
    id: string;
    name: string;
    status: FunctionRunStatus;
  }[]
}

export default function FunctionList({ functions }: FunctionListProps) {

  return (
    <>
      {functions.length < 1 ? (
        <p className="text-slate-600">No functions called</p>
      ) : (
        <ul className="flex flex-col space-y-4">
          {functions.map((func) => {
            const itemStatus = statusStyles(func.status);
            return (
              <div key={func.id} className="flex items-center gap-2">
                <itemStatus.icon />
                <span>{func.name}</span>
              </div>
            );
          })}
        </ul>
      )}
    </>
  );
}
