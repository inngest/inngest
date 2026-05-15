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
import ArchiveEventModal from '../Events/ArchiveEventModal';
import { SendEventModal } from '../Events/SendEventModal';

export const ActionsMenu = (row: Row<EventType>) => {
  const { isArchived } = useEnvironment();

  const [isSendEventModalVisible, setIsSendEventModalVisible] = useState(false);
  const [isArchiveEventModalVisible, setIsArchiveEventModalVisible] =
    useState(false);
  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            kind="secondary"
            appearance="outlined"
            size="small"
            icon={<RiMoreFill />}
          />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <OptionalTooltip
            tooltip={
              isArchived && 'Cannot send events. Environment is archived.'
            }
          >
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
              }}
              onSelect={() => setIsSendEventModalVisible(true)}
              disabled={isArchived}
            >
              <RiPlayLine className="h-4 w-4" />
              Send test event
            </DropdownMenuItem>
          </OptionalTooltip>
          <OptionalTooltip
            tooltip={row.original.archived && 'Send event to unarchive it.'}
          >
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
              }}
              onSelect={() => setIsArchiveEventModalVisible(true)}
              disabled={row.original.archived}
              className="text-error"
            >
              <RiArchive2Line className="h-4 w-4" />
              {row.original.archived ? 'Unarchive event' : 'Archive event'}
            </DropdownMenuItem>
          </OptionalTooltip>
        </DropdownMenuContent>
      </DropdownMenu>

      <SendEventModal
        isOpen={isSendEventModalVisible}
        eventName={row.original.name}
        onClose={() => setIsSendEventModalVisible(false)}
      />
      <ArchiveEventModal
        isOpen={isArchiveEventModalVisible}
        eventName={row.original.name}
        onClose={() => setIsArchiveEventModalVisible(false)}
      />
    </>
  );
};
