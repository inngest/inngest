import {
  IconCircleStatusCheck,
  IconCircleStatusArrowPath,
  IconCircleStatusCross,
  IconCircleStatusMinus,
} from '@/icons';
import { FunctionRunStatus } from '@/store/generated';

export function statusStyles(status: FunctionRunStatus | null) {
  switch (status) {
    case FunctionRunStatus.Running:
      return {
        icon: IconCircleStatusArrowPath,
      };
    case FunctionRunStatus.Completed:
      return {
        icon: IconCircleStatusCheck,
      };
    case FunctionRunStatus.Failed:
      return {
        icon: IconCircleStatusCross,
      };
    default:
      return {
        icon: IconCircleStatusMinus,
      };
  }
}

export default function FunctionList({ row }) {
  const { functions } = row?.original;

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
                <itemStatus.icon className="h-6 w-6"/>
                <span>{func.name}</span>
              </div>
            );
          })}
        </ul>
      )}
    </>
  );
}
