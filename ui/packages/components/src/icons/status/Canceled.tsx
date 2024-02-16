import { cn } from '@inngest/components/utils/classNames';

export function IconStatusCanceled({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      className={cn('text-slate-500 dark:text-slate-300', className)}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <title>{title}</title>
      <path
        d="M14.5 12.3333H9.5M21 12.3333C21 17.3039 16.9706 21.3333 12 21.3333C7.02944 21.3333 3 17.3039 3 12.3333C3 7.36277 7.02944 3.33333 12 3.33333C16.9706 3.33333 21 7.36277 21 12.3333Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
