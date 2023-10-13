import { classNames } from '@inngest/components/utils/classNames';

import { IconCheckCircle, IconExclamationTriangle } from '@/icons';

type AppCardHeaderProps = {
  functionCount: number;
  connected: boolean;
};

export default function AppCardHeader({ connected, functionCount }: AppCardHeaderProps) {
  let headerColor, headerLabel, headerIcon;

  if (!connected) {
    headerColor = 'bg-rose-600/50';
    headerLabel = 'No Connection';
    headerIcon = <IconExclamationTriangle className="h-5 w-5 text-white" />;
  } else if (functionCount < 1) {
    headerColor = 'bg-orange-400/70';
    headerLabel = 'No Functions Found';
    headerIcon = <IconExclamationTriangle className="h-5 w-5 text-white" />;
  } else {
    headerColor = 'bg-teal-400/50';
    headerLabel = 'Connected';
    headerIcon = <IconCheckCircle className="h-5 w-5 text-white" />;
  }

  return (
    <header
      className={classNames(
        headerColor,
        `flex items-center justify-between gap-2 rounded-t-md px-6 py-2.5 capitalize text-white`
      )}
    >
      <div className="flex items-center gap-2 leading-7">
        {headerIcon}
        {headerLabel}
      </div>
    </header>
  );
}
