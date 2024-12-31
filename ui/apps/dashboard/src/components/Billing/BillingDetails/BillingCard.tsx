import { cn } from '@inngest/components/utils/classNames';

export default function BillingCard({
  heading,
  actions,
  className,
  children,
}: {
  heading: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn('border-subtle rounded-md border px-6 py-4', className)}>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-muted">{heading}</h2>
        {actions}
      </div>
      {children}
    </div>
  );
}
