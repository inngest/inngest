import cn from '@/utils/cn';

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
    <div className={cn('rounded-lg border border-slate-200 p-6', className)}>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-xl font-semibold">{heading}</h2>
        {actions}
      </div>
      {children}
    </div>
  );
}
