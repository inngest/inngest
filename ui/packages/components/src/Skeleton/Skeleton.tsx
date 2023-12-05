import { classNames } from '@inngest/components/utils/classNames';

export function Skeleton({ className }: { className?: string }) {
  return (
    <span
      className={classNames(
        className,
        'relative block overflow-hidden rounded-md before:absolute before:inset-0 before:-translate-x-full before:animate-[shimmer_2s_infinite] before:bg-gradient-to-r before:from-transparent before:via-slate-700/70 before:to-transparent'
      )}
    />
  );
}
