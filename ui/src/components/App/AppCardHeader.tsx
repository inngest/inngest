import classNames from '@/utils/classnames';
import { IconCheckCircle, IconExclamationTriangle } from '@/icons';
import Badge from '@/components/Badge';

type AppCardHeaderProps = {
  functionCount: number;
  sdkVersion: string;
  connected: boolean;
};

export default function AppCardHeader({
  connected,
  functionCount,
  sdkVersion,
}: AppCardHeaderProps) {
  let headerColor, headerLabel, headerIcon;

  if (!connected) {
    headerColor = 'bg-rose-600/50';
    headerLabel = 'No Connection';
    headerIcon = <IconExclamationTriangle />;
  } else if (functionCount < 1) {
    headerColor = 'bg-orange-400/70';
    headerLabel = 'No Functions Found';
    headerIcon = <IconExclamationTriangle />;
  } else {
    headerColor = 'bg-teal-400/50';
    headerLabel = 'Connected';
    headerIcon = <IconCheckCircle />;
  }

  return (
    <header
      className={classNames(
        headerColor,
        `text-white rounded-t-md px-6 py-2.5 capitalize flex gap-2 items-center justify-between`
      )}
    >
      <div className="flex items-center gap-2 leading-7">
        {headerIcon}
        {headerLabel}
      </div>
      {sdkVersion && <Badge>SDK {sdkVersion}</Badge>}
    </header>
  );
}
