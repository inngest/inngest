import { Link, type LinkProps } from '@inngest/components/Link';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiCheckboxCircleFill,
  RiErrorWarningFill,
  RiInformationFill,
  type RemixiconComponentType,
} from '@remixicon/react';

type Severity = 'error' | 'info' | 'success' | 'warning';

type SeveritySpecific = {
  icon: RemixiconComponentType;
  iconClassName: string;
  wrapperClassName: string;
  linkClassName: string;
};

const severityStyles = {
  error: {
    icon: RiErrorWarningFill,
    iconClassName: 'text-error',
    wrapperClassName: 'bg-error dark:bg-error/40 text-error',
    linkClassName:
      'text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense',
  },
  info: {
    icon: RiInformationFill,
    iconClassName: 'text-info',
    wrapperClassName: 'bg-info dark:bg-info/40 text-info',
    linkClassName:
      'text-info decoration-info  hover:text-secondary-2xIntense hover:decoration-secondary-2xIntense',
  },
  success: {
    icon: RiCheckboxCircleFill,
    iconClassName: 'text-success',
    wrapperClassName: 'bg-success dark:bg-success/40 text-success',
    linkClassName:
      'text-success decoration-success hover:text-primary-2xIntense hover:decoration-primary-2xIntense',
  },
  warning: {
    icon: RiErrorWarningFill,
    iconClassName: 'text-warning',
    wrapperClassName: 'bg-warning dark:bg-warning/40 text-warning',
    linkClassName:
      'text-warning decoration-warning hover:text-accent-2xIntense hover:decoration-accent-2xIntense',
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

  /**
   * Additional button CTA.
   */
  button?: React.ReactNode;

  /**
   * Additional link CTA.
   */
  link?: React.ReactNode;
};

export function Alert({ children, className, severity, showIcon = true, button, link }: Props) {
  const Icon = severityStyles[severity].icon;

  return (
    <div
      className={cn('rounded-md px-4 py-3', severityStyles[severity].wrapperClassName, className)}
    >
      <div className="flex items-start gap-2 ">
        {showIcon && (
          <Icon className={cn('h-5 w-5 shrink-0', severityStyles[severity].iconClassName)} />
        )}

        <div className="leading-5">{children}</div>
      </div>
      {button && <div className={cn('mt-4', showIcon && 'ml-7')}>{button}</div>}
      {link}
    </div>
  );
}

function AlertLink({
  href,
  severity,
  children,
  ...props
}: React.PropsWithChildren<LinkProps & { severity: Severity }>) {
  const styles = severityStyles[severity].linkClassName;
  return (
    <Link href={href} {...props} className={cn(styles, props.className)}>
      {children}
    </Link>
  );
}

Alert.Link = AlertLink;
