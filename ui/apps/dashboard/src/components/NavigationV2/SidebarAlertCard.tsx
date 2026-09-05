import { cn } from '@inngest/components/utils/classNames';

const kindStyles = {
  default: 'border-subtle bg-canvasBase text-basis',
  error: 'border-tertiary-xSubtle bg-error text-error',
  warning: 'border-accent-xSubtle bg-warning text-warning',
} as const;

export type SidebarAlertCardProps = {
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
  footer?: React.ReactNode;
  footerClassName?: string;
  kind?: keyof typeof kindStyles;
};

export function SidebarAlertCard({
  children,
  className,
  contentClassName,
  footer,
  footerClassName,
  kind = 'default',
}: SidebarAlertCardProps) {
  return (
    <section
      className={cn(
        'w-full overflow-hidden rounded border leading-tight',
        kindStyles[kind],
        className,
      )}
    >
      <div className={cn('px-2 py-3', contentClassName)}>{children}</div>
      {footer && (
        <div
          className={cn('border-t border-inherit px-2 py-1.5', footerClassName)}
        >
          {footer}
        </div>
      )}
    </section>
  );
}
