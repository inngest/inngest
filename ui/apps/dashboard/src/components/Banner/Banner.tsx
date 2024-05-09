import { Button } from '@inngest/components/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiCloseLine, RiErrorWarningLine, RiInformationLine } from '@remixicon/react';

type Severity = 'info' | 'error' | 'warning';

const backgroundColors = {
  info: 'bg-blue-100',
  error: 'bg-rose-100',
  warning: 'bg-amber-100',
} as const satisfies { [key in Severity]: string };

const icons = {
  info: <RiInformationLine className="h-6 w-6 text-blue-700" />,
  error: <RiErrorWarningLine className="h-6 w-6 text-rose-700" />,
  warning: <RiErrorWarningLine className="h-6 w-6 text-amber-700" />,
} as const satisfies { [key in Severity]: React.ReactNode };

export function Banner({
  children,
  className,
  onDismiss,
  kind = 'info',
}: {
  children: React.ReactNode;
  className?: string;
  onDismiss?: () => void;
  kind?: Severity;
}) {
  const icon = icons[kind];
  const color = backgroundColors[kind];

  return (
    <div
      className={cn(
        className,
        color,
        'flex w-full items-center justify-between px-2 py-2 md:px-4 lg:px-8'
      )}
    >
      <div className="flex items-start gap-1 text-sm">
        <span className="shrink-0">{icon}</span>
        <span className="leading-6">{children}</span>
      </div>
      {onDismiss && (
        <Button
          size="small"
          appearance="text"
          btnAction={onDismiss}
          icon={<RiCloseLine className="h-5 w-5" />}
        />
      )}
    </div>
  );
}
