'use client';

import { useEffect, useState } from 'react';
import * as Progress from '@radix-ui/react-progress';

import { cn } from '../utils/classNames';

type ProgressBarProps = {
  limit: number | null;
  value: number;
  overageAllowed?: boolean;
};

const ProgressBar = ({ limit, value, overageAllowed }: ProgressBarProps) => {
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    const calculatedProgress = limit === null ? 0 : Math.min((value / limit) * 100, 100);
    const timer = setTimeout(() => setProgress(calculatedProgress), 500);
    return () => clearTimeout(timer);
  }, []);

  const isOverTheLimit = limit !== null && value > limit;

  return (
    <Progress.Root
      className={cn(
        'bg-canvasMuted relative h-6 overflow-hidden rounded-md',
        limit === null && 'bg-secondary-subtle'
      )}
      style={{
        transform: 'translateZ(0)',
      }}
      value={progress}
    >
      <Progress.Indicator
        className={cn(
          'ease-[cubic-bezier(0.65, 0, 0.35, 1)] bg-primary-moderate size-full transition-transform duration-700',
          isOverTheLimit && overageAllowed && 'bg-accent-subtle',
          isOverTheLimit && !overageAllowed && 'bg-errorContrast'
        )}
        style={{ transform: `translateX(-${100 - progress}%)` }}
      />
    </Progress.Root>
  );
};

export default ProgressBar;
