import { Fragment, type ReactElement } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Skeleton } from '@inngest/components/Skeleton';

import type { SchemaEntry } from './SchemasContext/types';

type Props = {
  entries: SchemaEntry[];
  error: Error | null;
  isLoading: boolean; // initial load only
  isFetchingNextPage: boolean;
  hasFetchedMax: boolean;
  hasNextPage: boolean;
  fetchNextPage: () => void;
  renderEntry: (entry: SchemaEntry, preventExpand?: boolean) => ReactElement;
};

export function SchemaExplorerSwitcher({
  entries,
  error,
  isLoading,
  isFetchingNextPage,
  hasFetchedMax,
  hasNextPage,
  fetchNextPage,
  renderEntry,
}: Props): ReactElement {
  const showShimmers = isLoading || isFetchingNextPage;
  const showError = Boolean(error) && !showShimmers;

  return (
    <Fragment>
      <div className="flex flex-col gap-1">
        {entries.map((entry) => renderEntry(entry))}
      </div>
      {showShimmers && <LoadingShimmers />}
      {showError && (
        <Alert className="mt-2 text-xs" severity="error">
          <div className="flex flex-row gap-2">
            <div>Failed to load schemas</div>
            {hasNextPage && (
              <Button
                appearance="outlined"
                kind="secondary"
                size="small"
                label="Retry"
                onClick={fetchNextPage}
              />
            )}
          </div>
        </Alert>
      )}
      {!showError && !showShimmers && hasFetchedMax && (
        <Alert severity="info" className="mt-2 text-xs">
          Please use search to target schemas.
        </Alert>
      )}
    </Fragment>
  );
}

function LoadingShimmers(): ReactElement {
  return (
    <div className="mt-2 flex flex-col gap-2">
      <Skeleton key={0} className="h-6" />
      <Skeleton key={1} className="h-6" />
      <Skeleton key={2} className="h-6" />
    </div>
  );
}
