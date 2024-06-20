import { cn } from '@inngest/components/utils/classNames';

export function IconStatusCompleted({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      className={cn('text-status-completed', className)}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <title>{title}</title>
      <path
        d="M9 13.5833L11.25 15.8333L15 10.5833M21 12.8333C21 17.8039 16.9706 21.8333 12 21.8333C7.02944 21.8333 3 17.8039 3 12.8333C3 7.86277 7.02944 3.83333 12 3.83333C16.9706 3.83333 21 7.86277 21 12.8333Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
