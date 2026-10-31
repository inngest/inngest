import { Button } from '@inngest/components/Button';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Search } from '@inngest/components/Forms/Search';
import { Table, TableBlankState, TextCell } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';
import { type SessionKey } from '@inngest/components/types/session';
import { RiExternalLinkLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

const columnHelper = createColumnHelper<SessionKey>();
const DOCS_URL = 'https://www.inngest.com/docs/features/events-triggers/sessions';

const columns = [
  columnHelper.accessor('sessionKey', {
    header: 'Session key',
    enableSorting: false,
    cell: ({ row }) => (
      <TextCell>
        <span className="font-mono">{row.original.sessionKey}</span>
      </TextCell>
    ),
  }),
  columnHelper.accessor('createdAt', {
    header: 'First seen',
    enableSorting: false,
    cell: ({ row }) => <Time value={row.original.createdAt} />,
  }),
];

type SessionKeysProps = {
  sessionKeys: SessionKey[];
  isLoading: boolean;
  search: string;
  error?: Error | null;
  onSearchChange: (value: string) => void;
  onSubmitSearch: (sessionKey: string) => void;
  onRefresh: () => void;
  onSelectSessionKey: (sessionKey: string) => void;
  getSessionKeyHref: (sessionKey: string) => string;
  // Optional "Load more" paging: shown when hasMore and onLoadMore are set.
  hasMore?: boolean;
  isLoadingMore?: boolean;
  onLoadMore?: () => void;
};

export function SessionKeys({
  sessionKeys,
  isLoading,
  search,
  error,
  onSearchChange,
  onSubmitSearch,
  onRefresh,
  onSelectSessionKey,
  getSessionKeyHref,
  hasMore = false,
  isLoadingMore = false,
  onLoadMore,
}: SessionKeysProps) {
  const trimmedSearch = search.trim();

  return (
    <div className="bg-canvasBase text-basis flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="flex flex-col gap-4 px-3 py-3">
        <form
          className="w-full max-w-[360px]"
          onSubmit={(e) => {
            e.preventDefault();
            if (trimmedSearch) {
              onSubmitSearch(trimmedSearch);
            }
          }}
        >
          <Search
            name="sessionKey"
            placeholder="Search by session key"
            value={search}
            maxLength={128}
            autoFocus
            className="w-full"
            onUpdate={onSearchChange}
          />
        </form>
      </div>
      <div className="flex-1 overflow-y-auto">
        {error ? (
          <ErrorCard error={error} reset={onRefresh} />
        ) : (
          <>
            <Table
              columns={columns}
              data={sessionKeys}
              isLoading={isLoading}
              blankState={
                <TableBlankState
                  icon={<SessionsIcon />}
                  actions={
                    <Button
                      label="Go to docs"
                      href={DOCS_URL}
                      target="_blank"
                      icon={<RiExternalLinkLine />}
                      iconSide="left"
                    />
                  }
                  title={search ? `No session key found for "${search}"` : 'No sessions found'}
                  description="Session keys appear here after events are sent with session meta."
                />
              }
              onRowClick={(row) => onSelectSessionKey(row.original.sessionKey)}
              getRowHref={(row) => getSessionKeyHref(row.original.sessionKey)}
            />
            {hasMore && onLoadMore && (
              <div className="flex flex-col items-center py-2">
                <Button
                  kind="secondary"
                  appearance="outlined"
                  label="Load more"
                  onClick={onLoadMore}
                  loading={isLoadingMore}
                  disabled={isLoadingMore}
                />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
