'use client';

import type { ReactElement } from 'react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiDeleteBinLine, RiShare2Line } from '@remixicon/react';

import type { InsightsQueryStatement } from '@/gql/graphql';
import { useStoredQueries } from './QueryHelperPanel/StoredQueriesContext';

type QueryActionsMenuProps = {
  onOpenChange?: (open: boolean) => void;
  onSelectDelete: (query: InsightsQueryStatement) => void;
  open?: boolean;
  query: InsightsQueryStatement | undefined;
  trigger: ReactElement;
};

export function QueryActionsMenu({
  onOpenChange,
  onSelectDelete,
  open,
  query,
  trigger,
}: QueryActionsMenuProps) {
  const { shareQuery } = useStoredQueries();

  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {!query?.shared && (
          <DropdownMenuItem
            className="text-basis px-4"
            onSelect={() => {
              if (query === undefined) return;
              shareQuery(query.id);
            }}
          >
            <RiShare2Line className="size-4" />
            <span>Share with your org</span>
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          className="text-error px-4"
          onSelect={() => {
            if (query === undefined) return;
            onSelectDelete(query);
          }}
        >
          <RiDeleteBinLine className="size-4" />
          <span>Delete query</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
