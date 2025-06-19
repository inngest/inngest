import { RiLightbulbLine } from '@remixicon/react';

import { Button } from '../Button';
import { Pill } from '../Pill';
import { StatusDot } from '../Status/StatusDot';
import { getStatusTextClass } from '../Status/statusClasses';
import { cn } from '../utils/classNames';

export type StepHistoryProps = {
  dateStarted: Date;
  status: string;
  tagCount: number;
};

export const StepHistory = ({ dateStarted, status, tagCount }: StepHistoryProps) => (
  <div className="border-muted flex flex-row items-center justify-between gap-2 border-b px-4 py-3">
    <div className="text-muted w-[30%] whitespace-nowrap text-sm">
      {dateStarted.toLocaleString()}
    </div>

    <div
      className={cn('flex w-[30%] flex-row items-center gap-2 text-sm', getStatusTextClass(status))}
    >
      <StatusDot status={status} className="h-2.5 w-2.5" />
      {status}
    </div>
    <div className="w-[20%]">
      <Pill appearance="outlined" kind="primary">
        <div className="flex flex-row items-center gap-1">
          <RiLightbulbLine className="text-muted h-2.5 w-2.5" />
          {tagCount}
        </div>
      </Pill>
    </div>

    <div className="flex w-[20%] justify-end">
      <Button
        kind="secondary"
        appearance="outlined"
        size="small"
        label="View version"
        className="text-muted text-xs"
      />
    </div>
  </div>
);
