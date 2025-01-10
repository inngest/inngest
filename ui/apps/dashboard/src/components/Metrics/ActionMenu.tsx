'use client';

import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { Switch } from '@inngest/components/Switch';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { RiSettingsLine } from '@remixicon/react';

export const AUTO_REFRESH_INTERVAL = 5;

export const MetricsActionMenu = () => {
  const [autoRefresh, setAutoRefresh, removeAutoRefresh] = useSearchParam('autoRefresh');

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          kind="secondary"
          appearance="outlined"
          size="medium"
          icon={<RiSettingsLine />}
          className="text-sm"
        />
      </DropdownMenuTrigger>
      <div className="relative">
        <DropdownMenuContent align="end">
          <DropdownMenuItem className="hover:bg-canvasBase">
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
          </DropdownMenuItem>
        </DropdownMenuContent>
      </div>
    </DropdownMenu>
  );
};
