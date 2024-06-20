import { cn } from '@inngest/components/utils/classNames';

export function IconStatusFailed({ className, title }: { className?: string; title?: string }) {
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
        d="M9.75 9.91666L14.25 14.4167M14.25 9.91666L9.75 14.4167M21 12.1667C21 17.1372 16.9706 21.1667 12 21.1667C7.02944 21.1667 3 17.1372 3 12.1667C3 7.19609 7.02944 3.16666 12 3.16666C16.9706 3.16666 21 7.19609 21 12.1667Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
