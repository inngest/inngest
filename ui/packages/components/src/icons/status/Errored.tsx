import { cn } from '@inngest/components/utils/classNames';

export function IconStatusErrored({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      className={cn('text-status-failed', className)}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <title>{title}</title>
      <path
        d="M11.9998 9.00006V12.7501M11.9998 15.7501H12.0073V15.7576H11.9998V15.7501ZM21 12C21 16.9706 16.9706 21 12 21C7.02944 21 3 16.9706 3 12C3 7.02944 7.02944 3 12 3C16.9706 3 21 7.02944 21 12Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
