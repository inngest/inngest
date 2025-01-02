import { cn } from '@inngest/components/utils/classNames';

export function Skeleton({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'bg-canvasMuted/20 relative block overflow-hidden rounded-sm',
        'before:via-canvasMuted/30 before:absolute before:inset-0 before:-translate-x-full before:animate-[shimmer_2s_infinite] before:bg-gradient-to-r before:from-transparent before:to-transparent',
        className
      )}
    />
  );
}
