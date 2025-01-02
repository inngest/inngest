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
  showArchive: () => void;
  showValidate: () => void;
  disableArchive?: boolean;
  disableValidate?: boolean;
};

export const ActionsMenu = ({
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
      <DropdownMenuContent side="bottom" align="end">
        <DropdownMenuItem>
          <OptionalTooltip tooltip={disableValidate && 'No syncs. App health check not available.'}>
            <Button
              disabled={disableValidate}
              appearance="ghost"
              kind="secondary"
              size="small"
              icon={<RiFirstAidKitLine className="h-4 w-4" />}
              iconSide="left"
              label="Check app health"
              className={`text-muted m-0 w-full justify-start ${
                disableValidate && 'cursor-not-allowed'
              }`}
              onClick={showValidate}
            />
          </OptionalTooltip>
        </DropdownMenuItem>

        {!isArchived && (
          <DropdownMenuItem>
            <OptionalTooltip
              tooltip={disableArchive && 'Parent app is archived. Archive action not available.'}
            >
              <Button
                appearance="ghost"
                kind="danger"
                size="small"
                icon={<RiArchive2Line className="h-4 w-4" />}
                iconSide="left"
                label={'Archive app'}
                className="m-0 w-full justify-start"
                onClick={showArchive}
              />
            </OptionalTooltip>
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
