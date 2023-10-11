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
    iconClassName: 'text-red-500',
    wrapperClassName: 'bg-red-50 border-red-200 text-red-800',
  },
  info: {
    icon: InformationCircleIcon,
    iconClassName: '',
    wrapperClassName: 'bg-indigo-50 border-indigo-100 text-indigo-600',
  },
  warning: {
    icon: ExclamationTriangleIcon,
    iconClassName: 'text-amber-500',
    wrapperClassName: 'bg-amber-50 border-amber-600/30 text-amber-900',
  },
} as const satisfies { [key in Severity]: SeveritySpecific };

type Props = {
  children: React.ReactNode;
  className?: string;
  severity: Severity;
};

export function Alert({ children, className, severity }: Props) {
  const Icon = severityStyles[severity].icon;

  return (
    <div
      className={cn(
        'flex items-start gap-2 rounded-md border px-4 py-3 text-sm font-medium',
        severityStyles[severity].wrapperClassName,
        className
      )}
    >
      <div>
        <Icon className={cn('w-5', severityStyles[severity].iconClassName)} />
      </div>

      <div>{children}</div>
    </div>
  );
}
