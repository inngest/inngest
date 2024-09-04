import { useState } from 'react';
import { RiArrowDownSFill, RiArrowRightSFill } from '@remixicon/react';

import { RangePicker } from '../DatePicker';
import EntityFilter from '../Filter/EntityFilter';
import { Pill } from '../Pill';
import { subtractDuration } from '../utils/date';
import { FunctionStatus } from './FunctionStatus';

type EntityType = {
  id: string;
  name: string;
};

type DateRange = {
  start?: Date;
  end?: Date;
  key?: string;
};

export const Dashboard = ({
  apps = [],
  functions = [],
  upgradeCutoff,
}: {
  apps?: EntityType[];
  functions?: EntityType[];
  upgradeCutoff: Date;
}) => {
  const [app, setApp] = useState<string[]>([]);
  const [fn, setFn] = useState<string[]>([]);
  const [timeRange, setTimeRange] = useState<DateRange>();
  const [overviewOpen, setOverviewOpen] = useState(true);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="bg-canvasBase flex h-16 w-full flex-row items-center justify-between px-3 py-5">
        <div className="flex flex-row items-center justify-start gap-x-2">
          <EntityFilter
            type="app"
            onFilterChange={setApp}
            selectedEntities={app}
            entities={apps}
            className="h-8"
          />
          <EntityFilter
            type="function"
            onFilterChange={setFn}
            selectedEntities={fn}
            entities={functions}
            className="h-8"
          />
        </div>
        <div className="flex flex-row items-center justify-end gap-x-2">
          <Pill appearance="outlined" kind="warning">
            <div className="text-nowrap">15m delay</div>
          </Pill>
          <RangePicker
            className="w-full"
            upgradeCutoff={upgradeCutoff}
            onChange={(range) =>
              setTimeRange(
                range.type === 'relative'
                  ? { start: subtractDuration(new Date(), range.duration), end: new Date() }
                  : { start: range.start, end: range.end }
              )
            }
          />
        </div>
      </div>
      <div className="px-6">
        <div className="bg-canvasSubtle item-start flex h-full w-full flex-col items-start">
          <div className="leading-non text-subtle my-4 flex w-full flex-row items-center justify-start gap-x-2 text-xs uppercase">
            {overviewOpen ? (
              <RiArrowDownSFill className="cursor-pointer" onClick={() => setOverviewOpen(false)} />
            ) : (
              <RiArrowRightSFill className="cursor-pointer" onClick={() => setOverviewOpen(true)} />
            )}
            <div>Overview</div>

            <hr className="border-subtle w-full" />
          </div>
          {overviewOpen && (
            <div className="flex flex-row items-center justify-start">
              <FunctionStatus />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
