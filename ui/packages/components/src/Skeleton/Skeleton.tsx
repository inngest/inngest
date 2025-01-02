import { cn } from '@inngest/components/utils/classNames';

export function Skeleton({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'before:via-canvasMuted/70 relative block overflow-hidden rounded-md before:absolute before:inset-0 before:-translate-x-full before:animate-[shimmer_2s_infinite] before:bg-gradient-to-r before:from-transparent before:to-transparent',
        className
      )}
    />
  );
}
