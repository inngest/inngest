import { Button } from '@inngest/components/Button';
import { Link, type LinkProps } from '@inngest/components/Link';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiCheckboxCircleLine,
  RiCloseLine,
  RiErrorWarningLine,
  RiInformationLine,
  type RemixiconComponentType,
} from '@remixicon/react';

export type Severity = 'error' | 'info' | 'success' | 'warning';

type SeveritySpecific = {
  icon: RemixiconComponentType;
  iconClassName: string;
  wrapperClassName: string;
  linkClassName: string;
  borderStyles: string;
};

const severityStyles = {
  error: {
    icon: RiErrorWarningLine,
    iconClassName: 'text-error',
    wrapperClassName: 'bg-error dark:bg-error/40 text-error',
    linkClassName:
      'text-error decoration-error hover:text-tertiary-2xIntense hover:decoration-tertiary-2xIntense',
    borderStyles: 'border-tertiary-2xSubtle',
  },
  info: {
    icon: RiInformationLine,
    iconClassName: 'text-info',
    wrapperClassName: 'bg-info dark:bg-info/40 text-info',
    linkClassName:
      'text-info decoration-info  hover:text-secondary-2xIntense hover:decoration-secondary-2xIntense',
    borderStyles: 'border-secondary-2xSubtle',
  },
  success: {
    icon: RiCheckboxCircleLine,
    iconClassName: 'text-success',
    wrapperClassName: 'bg-success dark:bg-success/40 text-success',
    linkClassName:
      'text-success decoration-success hover:text-primary-2xIntense hover:decoration-primary-2xIntense',
    borderStyles: 'border-primary-2xSubtle',
  },
  warning: {
    icon: RiErrorWarningLine,
    iconClassName: 'text-warning',
    wrapperClassName: 'bg-warning dark:bg-warning/40 text-warning',
    linkClassName:
      'text-warning decoration-warning hover:text-accent-2xIntense hover:decoration-accent-2xIntense',
    borderStyles: 'border-accent-2xSubtle',
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
   * Ability to close the banner.
   */
  onDismiss?: () => void;

  /**
   * Additional link or button CTA.
   */
  cta?: React.ReactNode;
};

export function Banner({
  children,
  className,
  onDismiss,
  severity = 'info',
  showIcon = true,
  cta,
}: Props) {
  const Icon = severityStyles[severity].icon;

  return (
    <div
      className={cn(
        className,
        severityStyles[severity].wrapperClassName,
        'flex w-full items-center justify-between px-4 py-2'
      )}
    >
      <div className="flex grow items-start gap-1 text-sm">
        {showIcon && (
          <span className="shrink-0">
            <Icon className={cn('h-5 w-5 shrink-0', severityStyles[severity].iconClassName)} />
          </span>
        )}
        <span className="grow leading-6">{children}</span>
      </div>
      {cta}
      {onDismiss && (
        <Button
          size="small"
          appearance="ghost"
          onClick={onDismiss}
          icon={<RiCloseLine className={cn('h-5 w-5', severityStyles[severity].iconClassName)} />}
        />
      )}
    </div>
  );
}

function BannerLink({
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

Banner.Link = BannerLink;

export function ContextualBanner({
  title,
  children,
  className,
  onDismiss,
  severity = 'info',
  cta,
}: Omit<Props, 'showIcon'> & {
  title: React.ReactNode | string;
}) {
  return (
    <div
      className={cn(className, severityStyles[severity].wrapperClassName, 'flex w-full flex-col ')}
    >
      <div
        className={cn(
          'flex grow items-center justify-between gap-1 border-b px-4 py-2',
          severityStyles[severity].borderStyles
        )}
      >
        <span className="grow text-sm leading-6">{title}</span>
        {cta}
        {onDismiss && (
          <Button
            size="small"
            appearance="ghost"
            onClick={onDismiss}
            icon={<RiCloseLine className={cn('h-5 w-5', severityStyles[severity].iconClassName)} />}
          />
        )}
      </div>

      <div className="flex grow items-start gap-1 text-sm">
        <span className="grow leading-6">{children}</span>
      </div>
    </div>
  );
}

function ContextualList({ children, ...props }: React.PropsWithChildren) {
  return (
    <ul {...props} className="list-outside list-disc py-2 pl-6">
      {children}
    </ul>
  );
}

ContextualBanner.Link = BannerLink;
ContextualBanner.List = ContextualList;
