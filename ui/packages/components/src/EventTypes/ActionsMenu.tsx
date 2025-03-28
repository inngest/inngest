'use client';

import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiArchive2Line, RiMoreFill, RiPlayLine } from '@remixicon/react';

export type AppActions = {
  isArchived: boolean;
};

export const ActionsMenu = ({ isArchived }: AppActions) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button kind="secondary" appearance="ghost" size="small" icon={<RiMoreFill />} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {/* TODO: implement onSelect action */}
        <DropdownMenuItem onSelect={() => {}}>
          <RiPlayLine className="h-4 w-4" />
          Send test event
        </DropdownMenuItem>

        {/* TODO: implement onSelect action */}
        <DropdownMenuItem onSelect={() => {}} className="text-error">
          <RiArchive2Line className="h-4 w-4" />
          {isArchived ? 'Unarchive event' : 'Archive event'}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
