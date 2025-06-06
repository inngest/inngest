import { cn } from '../utils/classNames';

type GroupSpanProps = {
  depth: number;
  width: number;
};

export const GroupSpan = ({ width, depth = 1 }: GroupSpanProps) => {
  return (
    <div
      className={cn(
        `bg-btnPrimary absolute rounded-sm`,
        depth === 1 ? 'bg-opacity-20' : 'bg-opacity-10'
      )}
      style={{
        width: width,
        height: 'calc(100% - 8px)',
      }}
    />
  );
};
