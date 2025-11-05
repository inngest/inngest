'use client';

import { Fragment, type ReactElement } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/Button';
import { Skeleton } from '@inngest/components/Skeleton';

import type { SchemaEntry } from './SchemasContext/types';

type Props = {
  entries: SchemaEntry[];
  error: Error | null;
  isLoading: boolean; // initial load only
  isFetchingNextPage: boolean;
  hasNextPage: boolean;
  fetchNextPage: () => void;
  renderEntry: (entry: SchemaEntry, idx: number) => ReactElement;
};

export function SchemaExplorerSwitcher({
  entries,
  error,
  isLoading,
  isFetchingNextPage,
  hasNextPage,
  fetchNextPage,
  renderEntry,
}: Props): ReactElement {
  const showShimmers = isLoading || isFetchingNextPage;
  const showError = Boolean(error) && !showShimmers;

  return (
    <Fragment>
      <div className="flex flex-col gap-1">{entries.map(renderEntry)}</div>
      {showShimmers && <LoadingShimmers />}
      {showError && (
        <Alert
          button={
            hasNextPage ? (
              <Button
                appearance="outlined"
                kind="secondary"
                size="small"
                label="Retry"
                onClick={fetchNextPage}
              />
            ) : undefined
          }
          severity="error"
        >
          Failed to load schemas
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
