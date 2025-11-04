'use client';

import type { ReactElement } from 'react';

import type { SchemaEntry } from './SchemasContext/types';

type Props = {
  entries: SchemaEntry[];
  error: string | null;
  isLoading: boolean;
  isFetchingNextPage: boolean;
  renderEntry: (entry: SchemaEntry, idx: number) => ReactElement;
};

export function SchemaExplorerSwitcher({
  entries,
  error,
  isLoading,
  isFetchingNextPage,
  renderEntry,
}: Props): ReactElement {
  const hasRemoteEntries = entries.some((e) => !e.isShared);

  if (error && !hasRemoteEntries) {
    return (
      <>
        <div className="flex flex-col gap-1">
          {entries.filter((e) => e.isShared).map(renderEntry)}
        </div>
        <div className="text-danger mb-2 mt-2 text-sm">Failed to load custom schemas</div>
      </>
    );
  }

  if (!isLoading) {
    return (
      <>
        <div className="flex flex-col gap-1">{entries.map(renderEntry)}</div>
        {isFetchingNextPage && <LoadingShimmers />}
        {/* Non-blocking error while loading more */}
        {error && hasRemoteEntries && (
          <div className="text-warning mt-2 text-sm">Failed to load additional schemas</div>
        )}
      </>
    );
  }

  return <LoadingShimmers />;
}

function LoadingShimmers(): ReactElement {
  return (
    <div className="mt-2 flex flex-col gap-2">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="bg-canvasSubtle h-6 animate-pulse rounded" />
      ))}
    </div>
  );
}
