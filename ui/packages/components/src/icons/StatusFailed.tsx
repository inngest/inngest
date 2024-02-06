import { cn } from '@inngest/components/utils/classNames';

export function IconStatusFailed({ className, title }: { className?: string; title?: string }) {
  return (
    <svg
      className={cn('text-red-500', className)}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <title>{title}</title>
      <path
        d="M11.9998 8.99994V12.7499M2.69653 16.1256C1.83114 17.6256 2.91371 19.4999 4.64544 19.4999H19.3541C21.0858 19.4999 22.1684 17.6256 21.303 16.1256L13.9487 3.37807C13.0828 1.87723 10.9167 1.87723 10.0509 3.37807L2.69653 16.1256ZM11.9998 15.7499H12.0073V15.7574H11.9998V15.7499Z"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}
