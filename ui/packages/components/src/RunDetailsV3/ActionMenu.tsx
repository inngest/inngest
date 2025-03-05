'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiArrowDownSLine, RiCloseCircleLine } from '@remixicon/react';

export type RunActions = {
  cancel: () => void;
  reRun: () => void;
  allowCancel?: boolean;
};

export const ActionsMenu = ({ cancel, reRun, allowCancel }: RunActions) => {
  const [open, setOpen] = useState(false);
  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger className="h-7" />
      <Button
        kind="primary"
        appearance="outlined"
        size="medium"
        split={true}
        icon={
          <RiArrowDownSLine
            className="transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180"
            onClick={(e) => {
              setOpen(!open);
              e.stopPropagation();
            }}
          />
        }
        iconSide="right"
        label="Rerun"
        className="group text-sm"
        onClick={reRun}
      />

      <DropdownMenuContent align="start">
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
