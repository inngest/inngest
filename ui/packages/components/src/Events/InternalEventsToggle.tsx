import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { Switch, SwitchWrapper } from '@inngest/components/Switch';
import { RiSettingsLine } from '@remixicon/react';

import { useBooleanSearchParam } from '../hooks/useSearchParam';

export const InternalEventsToggle = () => {
  const [includeInternalEvents, setIncludeInternalEvents, remove] =
    useBooleanSearchParam('includeInternal');
  const handleToggle = (include: boolean) => {
    if (include) {
      remove();
    } else {
      setIncludeInternalEvents(false);
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
        <DropdownMenuItem className="hover:bg-canvasBase">
          <SwitchWrapper>
            <div className="text-basis text-sm">Show internal events</div>
            <Switch
              id="show-internal-events"
              checked={includeInternalEvents ?? true}
              onCheckedChange={handleToggle}
            />
          </SwitchWrapper>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
