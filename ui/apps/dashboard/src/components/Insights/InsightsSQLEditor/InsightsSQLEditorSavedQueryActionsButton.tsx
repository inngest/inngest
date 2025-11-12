'use client';

import { useMemo } from 'react';
import { Button } from '@inngest/components/Button/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiDeleteBinLine, RiMore2Fill, RiShare2Line } from '@remixicon/react';

import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';
import type { Tab } from '../types';

type InsightsSQLEditorSavedQueryActionsButtonProps = { tab: Tab };

export function InsightsSQLEditorSavedQueryActionsButton({
  tab,
}: InsightsSQLEditorSavedQueryActionsButtonProps) {
  const { deleteQuery, queries } = useStoredQueries();

  const savedQuery = useMemo(() => {
    if (tab.savedQueryId === undefined) return undefined;

    return queries.data?.find((q) => q.id === tab.savedQueryId);
  }, [queries.data, tab.savedQueryId]);

  if (savedQuery === undefined) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button appearance="outlined" icon={<RiMore2Fill />} kind="secondary" size="medium" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {!savedQuery.shared && (
          <DropdownMenuItem
            className="text-basis px-4"
            onSelect={(e) => {
              e.preventDefault();
              // Placeholder: share action not implemented yet.
            }}
          >
            <RiShare2Line className="size-4" />
            <span>Share with your org</span>
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          className="text-error px-4"
          onSelect={() => {
            deleteQuery(savedQuery.id);
          }}
        >
          <RiDeleteBinLine className="size-4" />
          <span>Delete query</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
