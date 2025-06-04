import { cn } from '../utils/classNames';

type GroupSpanProps = {
  depth: number;
  width: number;
};

export const GroupSpan = ({ width, depth = 1 }: GroupSpanProps) => {
  return (
    <div
      className={cn(
        `bg-btnPrimary absolute z-0 rounded-sm`,
        depth === 1 ? 'bg-opacity-5' : 'bg-opacity-10'
      )}
      style={{
        width: width,
        height: 'calc(100% - 8px)',
      }}
    />
  );
};
