import { Button } from '@inngest/components/Button/NewButton';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { Switch, SwitchWrapper } from '@inngest/components/Switch';
import { RiSettingsLine } from '@remixicon/react';

import { useBooleanSearchParam } from '../hooks/useNewSearchParams';

export type EventsActionMenuProps = {
  setAutoRefresh?: () => void;
  autoRefresh?: boolean;
  intervalSeconds?: number;
};

export const EventsActionMenu = ({
  autoRefresh,
  setAutoRefresh,
  intervalSeconds = 5,
}: EventsActionMenuProps) => {
  const [includeInternalEvents, setIncludeInternalEvents, remove] =
    useBooleanSearchParam('includeInternal');
  const handleToggle = (include: boolean) => {
    if (include) {
      setIncludeInternalEvents(true);
    } else {
      remove();
    }
  };

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
      <DropdownMenuContent align="end">
        {setAutoRefresh && (
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
        )}
        <DropdownMenuItem className="hover:bg-canvasBase justify-between">
          <SwitchWrapper className="flex-1 justify-between">
            <div className="text-basis text-sm">Show internal events</div>
            <Switch
              id="show-internal-events"
              checked={includeInternalEvents ?? false}
              onCheckedChange={handleToggle}
              onClick={(e) => {
                e.stopPropagation();
              }}
            />
          </SwitchWrapper>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
