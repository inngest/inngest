'use client';

import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiArchive2Line, RiFirstAidKitLine, RiMore2Line } from '@remixicon/react';

export type AppActions = {
  isArchived: boolean;
  showUnarchive?: boolean;
  showArchive: () => void;
  showValidate: () => void;
  disableArchive?: boolean;
  disableValidate?: boolean;
};

export const ActionsMenu = ({
  showUnarchive = true,
  isArchived,
  showArchive,
  showValidate,
  disableArchive = false,
  disableValidate = false,
}: AppActions) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button kind="primary" appearance="outlined" size="medium" icon={<RiMore2Line />} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem disabled={disableValidate} onSelect={showValidate}>
          <OptionalTooltip tooltip={disableValidate && 'No syncs. App health check not available.'}>
            <RiFirstAidKitLine className="h-4 w-4" />
            Check app health
          </OptionalTooltip>
        </DropdownMenuItem>

        {(!isArchived || showUnarchive) && (
          <DropdownMenuItem disabled={disableArchive} onSelect={showArchive} className="text-error">
            <OptionalTooltip
              tooltip={disableArchive && 'Parent app is archived. Archive action not available.'}
            >
              <RiArchive2Line className="h-4 w-4" />
              {isArchived ? 'Unarchive app' : 'Archive app'}
            </OptionalTooltip>
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
