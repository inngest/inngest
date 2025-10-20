import { Button } from '@inngest/components/Button/NewButton';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiSettingsLine } from '@remixicon/react';

import { FeatureToggle } from '../FeatureToggle/FeatureToggle';
import { Switch } from '../Switch';

export type RunsActionMenuProps = {
  setAutoRefresh: () => void;
  setPreview: () => void;
  autoRefresh?: boolean;
  intervalSeconds?: number;
};

export const RunsActionMenu = ({
  autoRefresh,
  setAutoRefresh,
  setPreview,
  intervalSeconds = 5,
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
            <FeatureToggle
              defaultEnabled={false}
              featureFlagName="traces-preview"
              toggle={setPreview}
            />
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};
