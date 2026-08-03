import { cn } from '@inngest/components/utils/classNames';

const SHIMMER_DIRECTION_CLASSES = {
  right: 'before:-translate-x-full before:animate-[shimmer_2s_infinite] before:bg-gradient-to-r',
  left: 'before:translate-x-full before:animate-[shimmer-left_2s_infinite] before:bg-gradient-to-l',
  down: 'before:-translate-y-full before:animate-[shimmer-down_2s_infinite] before:bg-gradient-to-b',
  up: 'before:translate-y-full before:animate-[shimmer-up_2s_infinite] before:bg-gradient-to-t',
} as const;

type ShimmerDirection = keyof typeof SHIMMER_DIRECTION_CLASSES;

// `animate` defaults to true (the shimmer implies "this is actively
// loading") — pass false for a static placeholder, e.g. a confirmed-empty
// state where nothing is in flight and a shimmer would be misleading.
// `direction` picks which way the shimmer sweeps — 'right' (default,
// matching the original shimmer) or 'left'/'up'/'down' for a shape whose
// natural reading direction differs (e.g. 'up' for a single tall bar).
export function Skeleton({
  className,
  animate = true,
  direction = 'right',
}: {
  className?: string;
  animate?: boolean;
  direction?: ShimmerDirection;
}) {
  return (
    <span
      className={cn(
        'relative block overflow-hidden rounded-sm',
        animate
          ? `bg-canvasSubtle/20 before:via-canvasMuted/30 before:absolute before:inset-0 before:from-transparent before:to-transparent ${SHIMMER_DIRECTION_CLASSES[direction]}`
          : 'bg-canvasSubtle',
        className
      )}
    />
  );
}
