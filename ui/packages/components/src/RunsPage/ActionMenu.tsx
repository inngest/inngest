'use client';

import { Listbox } from '@headlessui/react';
import { NewButton } from '@inngest/components/Button';
import { RiSettingsLine } from '@remixicon/react';

import { Switch } from '../Switch';

export type RunsActionMenuProps = {
  setAutoRefresh: () => void;
  autoRefresh?: boolean;
  intervalSeconds?: number;
};

export const RunsActionMenu = ({
  autoRefresh,
  setAutoRefresh,
  intervalSeconds = 5,
}: RunsActionMenuProps) => {
  return (
    <Listbox>
      <Listbox.Button as="div">
        <NewButton
          kind="secondary"
          appearance="outlined"
          size="medium"
          icon={<RiSettingsLine />}
          className="text-sm"
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase border-subtle shadow-tooltip absolute right-1 top-1 z-50 h-[52px] w-[247px] gap-y-0.5 rounded border shadow-2xl">
          <Listbox.Option
            className="text-muted mx-2 mt-2 flex cursor-pointer flex-row items-center justify-between text-[13px]"
            value="toggleAutoRefresh"
          >
            <div className="flex flex-col">
              <div className="text-basis text-sm">Auto Refresh</div>
              <div className="text-basis text-xs">
                Refreshes data every {intervalSeconds} seconds
              </div>
            </div>
            <Switch
              checked={autoRefresh}
              className="data-[state=checked]:bg-primary-moderate"
              onClick={(e) => {
                e.stopPropagation();
                setAutoRefresh();
              }}
            />
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
