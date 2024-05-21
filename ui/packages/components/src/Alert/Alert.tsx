import { cn } from '@inngest/components/utils/classNames';
import {
  RiErrorWarningLine,
  RiInformationLine,
  type RemixiconComponentType,
} from '@remixicon/react';

type Severity = 'error' | 'info' | 'warning';

type SeveritySpecific = {
  icon: RemixiconComponentType;
  iconClassName: string;
  wrapperClassName: string;
};

const severityStyles = {
  error: {
    icon: RiErrorWarningLine,
    iconClassName: 'text-rose-700 dark:text-white',
    wrapperClassName: 'bg-rose-100 dark:bg-rose-600/50 text-rose-700 dark:text-slate-300',
  },
  info: {
    icon: RiInformationLine,
    iconClassName: 'text-blue-700 dark:text-white',
    wrapperClassName: 'bg-blue-100 dark:bg-blue-600/50 text-blue-700 dark:text-slate-300',
  },
  warning: {
    icon: RiErrorWarningLine,
    iconClassName: 'text-amber-700 dark:text-white',
    wrapperClassName: 'bg-amber-100 dark:bg-amber-600/50 text-amber-700 dark:text-slate-300',
  },
} as const satisfies { [key in Severity]: SeveritySpecific };

type Props = {
  /**
   * The content of the alert.
   */
  children: React.ReactNode;

  /**
   * Additional class names to apply to the alert.
   */
  className?: string;

  /**
   * The severity of the alert.
   */
  severity: Severity;

  /**
   * Whether to show the icon for the alert.
   */
  showIcon?: boolean;
};

export function Alert({ children, className, severity, showIcon = true }: Props) {
  const Icon = severityStyles[severity].icon;

  return (
    <div
      className={cn(
        'flex items-start gap-2 rounded-md px-4 py-3',
        severityStyles[severity].wrapperClassName,
        className
      )}
    >
      {showIcon && (
        <Icon className={cn('h-5 w-5 shrink-0', severityStyles[severity].iconClassName)} />
      )}

      <div className="leading-5">{children}</div>
    </div>
  );
}
