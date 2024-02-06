import { cn } from '@inngest/components/utils/classNames';

export function IconStatusQueued({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      className={cn('text-orange-500', className)}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <title>{title}</title>
      <path
        d="M6.75 19.5H17.25"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M6.75 4.5H17.25"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M15.75 19.5V16.371C15.7499 15.9732 15.5918 15.5917 15.3105 15.3105L12 12L8.6895 15.3105C8.40818 15.5917 8.25008 15.9732 8.25 16.371V19.5"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M8.25 4.5V7.629C8.25008 8.02679 8.40818 8.40826 8.6895 8.6895L12 12L15.3105 8.6895C15.5918 8.40826 15.7499 8.02679 15.75 7.629V4.5"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}
