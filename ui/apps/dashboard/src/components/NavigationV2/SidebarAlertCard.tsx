import { cn } from '@inngest/components/utils/classNames';

export const boostedStyles = {
  card: 'bg-contrast border-contrast text-onContrast',
  icon: 'text-secondary-moderate',
  link: 'text-secondary-subtle hover:decoration-secondary-subtle',
};

const kindStyles = {
  default: 'border-subtle bg-canvasBase text-basis',
  error: 'border-tertiary-xSubtle bg-error text-error',
  warning: 'border-accent-xSubtle bg-warning text-warning',
  caution: 'border-accent-xSubtle bg-accent-3xSubtle text-warning',
  boost: boostedStyles.card,
} as const;

export type SidebarAlertCardProps = {
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
  footer?: React.ReactNode;
  kind?: keyof typeof kindStyles;
};

export function SidebarAlertCard({
  children,
  className,
  contentClassName,
  footer,
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
        <div className="border-t border-inherit px-2 py-1.5">{footer}</div>
      )}
    </section>
  );
}
