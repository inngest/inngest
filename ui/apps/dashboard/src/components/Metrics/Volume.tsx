import { useState } from 'react';
import { RiArrowDownSFill, RiArrowRightSFill } from '@remixicon/react';

import type { EntityType } from './Dashboard';
import { RunsThrougput } from './RunsThroughput';
import { StepsThroughput } from './StepsThroughput';

export type MetricsFilters = {
  from: Date;
  until?: Date;
  selectedApps?: string[];
  selectedFns?: string[];
  autoRefresh?: boolean;
  functions: EntityType[];
};

export const MetricsVolume = () => {
  const [volumeOpen, setVolumeOpen] = useState(true);

  return (
    <div className="bg-canvasSubtle item-start flex h-full w-full flex-col items-start">
      <div className="leading-non text-subtle my-4 flex w-full flex-row items-center justify-start gap-x-2 text-xs uppercase">
        {volumeOpen ? (
          <RiArrowDownSFill className="cursor-pointer" onClick={() => setVolumeOpen(false)} />
        ) : (
          <RiArrowRightSFill className="cursor-pointer" onClick={() => setVolumeOpen(true)} />
        )}
        <div>Volume</div>

        <hr className="border-subtle w-full" />
      </div>
      {volumeOpen && (
        <div className="relative flex w-full flex-row items-center justify-start gap-2 overflow-hidden">
          <RunsThrougput />
          <StepsThroughput />
        </div>
      )}
    </div>
  );
};
