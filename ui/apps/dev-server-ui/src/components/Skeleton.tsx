import classNames from '@/utils/classnames';

export default function Skeleton({ className }: { className?: string }) {
  return (
    <span
      className={classNames(
        className,
        'relative before:absolute before:inset-0 before:-translate-x-full before:animate-[shimmer_2s_infinite] before:bg-gradient-to-r before:from-transparent before:via-slate-700/70 before:to-transparent overflow-hidden rounded-md'
      )}
    />
  );
}
