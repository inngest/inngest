import ExclamationTriangleIcon from '@heroicons/react/24/outline/ExclamationTriangleIcon';
import InformationCircleIcon from '@heroicons/react/24/outline/InformationCircleIcon';
import XMarkIcon from '@heroicons/react/24/outline/XMarkIcon';
import { Button } from '@inngest/components/Button';
import { classNames } from '@inngest/components/utils/classNames';

export function Banner({
  children,
  className,
  onDismiss,
  kind,
}: {
  children: React.ReactNode;
  className?: string;
  onDismiss?: () => void;
  kind?: 'info' | 'error';
}) {
  let Icon: React.ReactNode;
  let color: string = '';
  if (kind == 'info') {
    Icon = <InformationCircleIcon className="h-6 w-6 text-blue-700" />;
    color = 'bg-blue-100';
  } else if (kind == 'error') {
    Icon = <ExclamationTriangleIcon className="h-6 w-6 text-rose-700" />;
    color = 'bg-rose-100';
  }

  return (
    <div
      className={classNames(
        className,
        color,
        'flex w-full items-center justify-between px-2 py-2 md:px-4 lg:px-8'
      )}
    >
      <div className="flex items-center gap-1 text-sm">
        {Icon}
        {children}
      </div>
      {onDismiss && (
        <Button
          size="small"
          appearance="text"
          btnAction={onDismiss}
          icon={<XMarkIcon className="h-5 w-5" />}
        />
      )}
    </div>
  );
}
