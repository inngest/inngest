import { cn } from '@inngest/components/utils/classNames';

export function IconStatusSleeping({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      className={cn('text-status-running', className)}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <title>{title}</title>
      <path
        d="M21.7519 15.002C20.597 15.484 19.3296 15.7501 18 15.7501C12.6152 15.7501 8.25 11.3849 8.25 6.0001C8.25 4.6705 8.51614 3.40306 8.99806 2.24815C5.47566 3.71796 3 7.19492 3 11.2501C3 16.6349 7.36522 21.0001 12.75 21.0001C16.8052 21.0001 20.2821 18.5244 21.7519 15.002Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
