'use client';

import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { IconReplay } from '@inngest/components/icons/Replay';
import { RiArrowDownSLine, RiCloseCircleLine } from '@remixicon/react';

export type RunActions = {
  cancel: () => void;
  reRun: () => void;
  allowCancel?: boolean;
};

export const ActionsMenu = ({ cancel, reRun, allowCancel }: RunActions) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          kind="primary"
          appearance="outlined"
          size="medium"
          icon={
            <RiArrowDownSLine className="transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
          }
          iconSide="right"
          label="All actions"
          className="group text-sm"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={reRun} className="text-primary outline-none">
          <IconReplay className="h-4 w-4" />
          Rerun
        </DropdownMenuItem>

        <DropdownMenuItem
          onSelect={cancel}
          className={`text-error ${!allowCancel && 'cursor-not-allowed'}`}
          disabled={!allowCancel}
        >
          <RiCloseCircleLine className="h-4 w-4" />
          Cancel
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
