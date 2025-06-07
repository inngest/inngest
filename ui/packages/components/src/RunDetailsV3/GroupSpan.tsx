import { getStatusBackgroundClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';

type GroupSpanProps = {
  depth: number;
  width: number;
  status: string;
};

export const GroupSpan = ({ width, status, depth = 1 }: GroupSpanProps) => {
  return (
    <div
      className={cn(
        `absolute rounded-sm`,
        getStatusBackgroundClass(status),
        depth === 1 ? 'bg-opacity-20' : 'bg-opacity-10'
      )}
      style={{
        width: width,
        height: 'calc(100% - 8px)',
      }}
    />
  );
};
