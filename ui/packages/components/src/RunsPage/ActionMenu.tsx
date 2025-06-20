'use client';

import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiSettingsLine } from '@remixicon/react';

import { Switch } from '../Switch';

export type RunsActionMenuProps = {
  setAutoRefresh: () => void;
  autoRefresh?: boolean;
  intervalSeconds?: number;

  toggleTracesPreview: () => void;
  tracesPreviewEnabled?: boolean;
};

export const RunsActionMenu = ({
  autoRefresh,
  setAutoRefresh,
  intervalSeconds = 5,
  tracesPreviewEnabled,
  toggleTracesPreview,
}: RunsActionMenuProps) => {
  return (
    <div>
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

        <DropdownMenuContent>
          <DropdownMenuItem className="hover:bg-canvasBase">
            <div className="flex flex-col">
              <div className="text-basis text-sm">Auto Refresh</div>
              <div className="text-basis text-xs">
                Refreshes data every {intervalSeconds} seconds
              </div>
            </div>
            <div className="flex-1" />
            <Switch
              checked={autoRefresh}
              className="data-[state=checked]:bg-primary-moderate"
              onClick={(e) => {
                e.stopPropagation();
                setAutoRefresh();
              }}
            />
          </DropdownMenuItem>
          <DropdownMenuItem className="hover:bg-canvasBase">
            <div className="flex flex-col">
              <div className="text-basis text-sm">Traces Preview</div>
              <div className="text-basis text-xs">Use a new Developer Preview mode of traces</div>
            </div>
            <div className="flex-1" />
            <Switch
              checked={tracesPreviewEnabled}
              className="data-[state=checked]:bg-primary-moderate"
              onClick={(e) => {
                e.stopPropagation();
                toggleTracesPreview();
              }}
            />
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};
