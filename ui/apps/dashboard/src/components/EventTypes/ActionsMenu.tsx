'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { type EventType } from '@inngest/components/types/eventType';
import { RiArchive2Line, RiMoreFill, RiPlayLine } from '@remixicon/react';
import { type Row } from '@tanstack/react-table';

import { useEnvironment } from '@/components/Environments/environment-context';
import { SendEventModal } from '../Events/SendEventModal';

export const ActionsMenu = (row: Row<EventType>) => {
  const { isArchived } = useEnvironment();

  const [isModalVisible, setIsModalVisible] = useState(false);
  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button kind="secondary" appearance="ghost" size="small" icon={<RiMoreFill />} />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <OptionalTooltip tooltip={isArchived && 'Cannot send events. Environment is archived'}>
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                setIsModalVisible(true);
              }}
              disabled={isArchived}
            >
              <RiPlayLine className="h-4 w-4" />
              Send test event
            </DropdownMenuItem>
          </OptionalTooltip>

          {/* TODO: implement onSelect action */}
          <DropdownMenuItem
            onSelect={(e) => {
              e.preventDefault();
            }}
            className="text-error"
          >
            <RiArchive2Line className="h-4 w-4" />
            {isArchived ? 'Unarchive event' : 'Archive event'}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <SendEventModal
        isOpen={isModalVisible}
        eventName={row.original.name}
        onClose={(e) => {
          e?.stopPropagation();
          setIsModalVisible(false);
        }}
      />
    </>
  );
};
