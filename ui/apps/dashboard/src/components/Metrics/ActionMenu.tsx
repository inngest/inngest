'use client';

import { Listbox } from '@headlessui/react';
import { NewButton } from '@inngest/components/Button';
import { Switch } from '@inngest/components/Switch';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { RiSettingsLine } from '@remixicon/react';

const AUTO_REFRESH_INTERVAL = 5;

export type RunsActionMenuProps = {};

export const MetricsActionMenu = ({}: RunsActionMenuProps) => {
  const [autoRefresh, setAutoRefresh, removeAutoRefresh] = useSearchParam('auto-refresh');

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
            className="text-subtle mx-2 mt-2 flex cursor-pointer flex-row items-center justify-between text-[13px]"
            value="toggleAutoRefresh"
          >
            <div className="flex flex-col">
              <div className="text-basis text-sm">Auto Refresh</div>
              <div className="text-basis text-xs">
                Refreshes data every {AUTO_REFRESH_INTERVAL} seconds
              </div>
            </div>
            <Switch
              checked={autoRefresh === 'true'}
              className="data-[state=checked]:bg-primary-moderate cursor-pointer"
              onClick={(e) => {
                e.stopPropagation();
                if (autoRefresh === 'true') {
                  removeAutoRefresh();
                } else {
                  setAutoRefresh('true');
                }
              }}
            />
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
