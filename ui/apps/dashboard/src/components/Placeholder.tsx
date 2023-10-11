import cn from '@/utils/cn';

const shimmer = `relative overflow-hidden before:absolute before:inset-0 before:-translate-x-full before:animate-[shimmer_1.5s_infinite] before:bg-gradient-to-r before:from-transparent before:via-white/10 before:to-transparent`;

/**
 * Renders an animated placeholder for creating loading skeletons
 *
 * You need to set the size and the background color at a minium.
 *
 * @param className string
 */
export default function Placeholder({ className = '' }: { className?: string }) {
  return <span className={cn('rounded-sm', shimmer, className)}></span>;
}
