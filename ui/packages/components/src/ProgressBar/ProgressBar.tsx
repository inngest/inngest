import * as Progress from '@radix-ui/react-progress';

import { cn } from '../utils/classNames';

type ProgressBarProps = {
  limit: number | null;
  value: number;
  overageAllowed?: boolean;
};

const ProgressBar = ({ limit, value, overageAllowed }: ProgressBarProps) => {
  const progress = limit === null ? 0 : Math.min((value / limit) * 100, 100);
  const includedWidth = limit !== null && progress === 100 ? (limit / value) * 100 : progress;
  const additionalWidth = progress >= 100 ? 100 - includedWidth : 0;
  const isOverTheLimit = limit !== null && value > limit;
  // const isUnderTheLimit = limit !== null && value < limit;

  return (
    <Progress.Root
      className={cn(
        'relative flex h-6 overflow-hidden rounded-md',
        'outline-subtle outline outline-1 -outline-offset-1'
      )}
      value={progress}
      max={100}
    >
      <Progress.Indicator
        className={cn(
          'bg-primary-moderate',
          isOverTheLimit && !overageAllowed && 'bg-errorContrast'
        )}
        style={{ width: `${includedWidth}%` }}
      />
      <Progress.Indicator
        className={cn(
          'bg-primary-2xSubtle',
          isOverTheLimit && !overageAllowed && 'bg-errorContrast'
        )}
        style={{ width: `${additionalWidth}%` }}
      />
    </Progress.Root>
  );
};

export default ProgressBar;
