import type { ForwardRefExoticComponent, SVGProps } from 'react';
import {
  ExclamationCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
} from '@heroicons/react/20/solid';

import cn from '@/utils/cn';

type Severity = 'error' | 'info' | 'warning';

type SeveritySpecific = {
  icon: ForwardRefExoticComponent<SVGProps<SVGSVGElement>>;
  iconClassName: string;
  wrapperClassName: string;
};

const severityStyles = {
  error: {
    icon: ExclamationCircleIcon,
    iconClassName: 'text-rose-700',
    wrapperClassName: 'bg-rose-100 text-rose-700',
  },
  info: {
    icon: InformationCircleIcon,
    iconClassName: 'text-blue-700',
    wrapperClassName: 'bg-blue-100 text-blue-700',
  },
  warning: {
    icon: ExclamationTriangleIcon,
    iconClassName: 'text-amber-700',
    wrapperClassName: 'bg-amber-100 text-amber-700',
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
      {showIcon && <Icon className={cn('w-5 shrink-0', severityStyles[severity].iconClassName)} />}

      <div className="leading-5">{children}</div>
    </div>
  );
}
