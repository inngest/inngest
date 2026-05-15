import type { ReactElement } from 'react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiAlignLeft, RiDeleteBinLine, RiShare2Line } from '@remixicon/react';

import type { InsightsQueryStatement } from '@/gql/graphql';
import { useSQLEditorInstance } from './InsightsSQLEditor/SQLEditorContext';
import { useStoredQueries } from './QueryHelperPanel/StoredQueriesContext';
import { isQuerySnapshot } from './queries';
import type { QuerySnapshot } from './types';

type QueryActionsMenuProps = {
  onOpenChange?: (open: boolean) => void;
  onSelectDelete: (query: InsightsQueryStatement | QuerySnapshot) => void;
  open?: boolean;
  query: InsightsQueryStatement | QuerySnapshot | undefined;
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

  // Try to get editor instance, returns null if context is not available (e.g., in sidebar)
  const editorInstance = useSQLEditorInstance();
  const editorRef = editorInstance?.editorRef ?? null;

  const handleFormatSQL = () => {
    if (!editorRef) return;
    const editor = editorRef.current;
    if (!editor) return;

    // Trigger the format document action
    editor.getAction('editor.action.formatDocument')?.run();
  };

  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {editorRef && (
          <DropdownMenuItem
            className="text-basis px-4 outline-none"
            onSelect={handleFormatSQL}
          >
            <RiAlignLeft className="size-4" />
            <span>Format SQL</span>
          </DropdownMenuItem>
        )}
        {isActualQueryAndUnshared(query) && (
          <DropdownMenuItem
            className="text-basis px-4 outline-none"
            onSelect={() => {
              if (query === undefined || isQuerySnapshot(query)) return;
              shareQuery(query.id);
            }}
          >
            <RiShare2Line className="size-4" />
            <span>Share with your org</span>
          </DropdownMenuItem>
        )}
        {query !== undefined && (
          <DropdownMenuItem
            className="text-error px-4 outline-none"
            onSelect={() => {
              onSelectDelete(query);
            }}
          >
            <RiDeleteBinLine className="size-4" />
            <span>Delete query</span>
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function isActualQueryAndUnshared(
  query: InsightsQueryStatement | QuerySnapshot | undefined,
) {
  return query !== undefined && !isQuerySnapshot(query) && !query.shared;
}
