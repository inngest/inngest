import * as Progress from '@radix-ui/react-progress';

import { cn } from '../utils/classNames';

export type ProgressBarProps = {
  limit: number | null;
  value: number;
  overageAllowed?: boolean;
  className?: string;
  kind?: 'default' | 'error';
  size?: 'default' | 'small';
};

const ProgressBar = ({
  limit,
  value,
  overageAllowed,
  className,
  kind = 'default',
  size = 'default',
}: ProgressBarProps) => {
  const progress = limit === null ? 0 : Math.min((value / limit) * 100, 100);
  const includedWidth = limit !== null && progress === 100 ? (limit / value) * 100 : progress;
  const additionalWidth = progress >= 100 ? 100 - includedWidth : 0;
  const isOverTheLimit = limit !== null && value > limit;
  // const isUnderTheLimit = limit !== null && value < limit;

  return (
    <Progress.Root
      className={cn(
        'relative flex overflow-hidden',
        size === 'default' && 'outline-subtle h-6 rounded-md outline outline-1 -outline-offset-1',
        size === 'small' && 'h-1 rounded-sm',
        kind === 'error' && 'bg-tertiary-xSubtle',
        className
      )}
      value={progress}
      max={100}
    >
      <Progress.Indicator
        className={cn(
          'bg-primary-moderate',
          isOverTheLimit && !overageAllowed && 'bg-errorContrast',
          kind === 'error' && 'bg-tertiary-intense'
        )}
        style={{ width: `${includedWidth}%` }}
      />
      <Progress.Indicator
        className={cn(
          'bg-primary-2xSubtle',
          isOverTheLimit && !overageAllowed && 'bg-errorContrast',
          kind === 'error' && 'bg-tertiary-intense'
        )}
        style={{ width: `${additionalWidth}%` }}
      />
    </Progress.Root>
  );
};

export default ProgressBar;
