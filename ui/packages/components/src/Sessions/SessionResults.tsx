import { useCallback, useDeferredValue, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { Search } from '@inngest/components/Forms/Search';
import { SelectGroup } from '@inngest/components/Select/Select';
import { Table, TableBlankState } from '@inngest/components/Table';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import { useBatchedSearchParams } from '@inngest/components/hooks/useSearchParams';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';
import { type Session } from '@inngest/components/types/session';
import { durationToString, parseDuration } from '@inngest/components/utils/date';
import { RiExternalLinkLine } from '@remixicon/react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, type LinkComponentProps } from '@tanstack/react-router';

import type { RangeChangeProps } from '../DatePicker/RangePicker';
import { useColumns } from './columns';

const DEFAULT_RANGE = '7d';
// TODO: Replace with the sessions docs URL and include a ref param for dashboard traffic.
const DOCS_URL = 'ui/packages/components/src/Sessions/SessionResults.tsx';

type RoutePath = LinkComponentProps['to'] | string;

type SessionResultsProps = {
  envID: string;
  // Max selectable range in days, from the account's history entitlement.
  maxRangeDays: number;
  sessionKey: string;
  last?: string;
  start?: string;
  end?: string;
  pathCreator: {
    session: (params: { sessionKey: string; sessionId: string }) => RoutePath;
    function: (params: { functionSlug: string }) => RoutePath;
  };
  getSessions: (params: {
    sessionKey: string;
    sessionIdSearch?: string;
    startTime: string;
    endTime: string;
  }) => Promise<Session[]>;
  onEditSearch: () => void;
};

export function SessionResults({
  envID,
  maxRangeDays,
  sessionKey,
  last,
  start,
  end,
  pathCreator,
  getSessions,
  onEditSearch,
}: SessionResultsProps) {
  const navigate = useNavigate();
  const columns = useColumns({ pathCreator });
  const [search, setSearch] = useState('');
  const trimmedSearch = search.trim();
  const deferredSearch = useDeferredValue(trimmedSearch);

  const batchUpdate = useBatchedSearchParams();

  const calculatedStartTime = useCalculatedStartTime({
    lastDays: last,
    startTime: start,
    defaultTime: DEFAULT_RANGE,
  });
  const startTime = calculatedStartTime.toISOString();
  const endTime = useMemo(() => (end ? new Date(end) : new Date()).toISOString(), [end]);

  const onDaysChange = useCallback(
    (value: RangeChangeProps) => {
      if (value.type === 'relative') {
        batchUpdate({
          last: durationToString(value.duration),
          start: null,
          end: null,
        });
      } else {
        batchUpdate({
          last: null,
          start: value.start.toISOString(),
          end: value.end.toISOString(),
        });
      }
    },
    [batchUpdate]
  );

  const { data, error, isPending, isFetching, refetch } = useQuery({
    queryKey: ['sessions', envID, sessionKey, deferredSearch, startTime, endTime],
    queryFn: () =>
      getSessions({
        sessionKey,
        sessionIdSearch: deferredSearch,
        startTime,
        endTime,
      }),
    refetchOnWindowFocus: false,
  });

  const sessions = useMemo(() => data ?? [], [data]);

  return (
    <div className="bg-canvasBase text-basis flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="flex flex-col gap-1.5 px-3 py-3 md:flex-row md:items-center">
        <SelectGroup>
          <span className="border-muted bg-modalBase text-muted box-content flex h-[24px] items-center rounded border px-2 text-xs">
            Time range
          </span>
          <TimeFilter
            className="rounded-l-none border-l-0"
            daysAgoMax={maxRangeDays}
            onDaysChange={onDaysChange}
            defaultValue={
              last
                ? { type: 'relative', duration: parseDuration(last) }
                : start && end
                ? {
                    type: 'absolute',
                    start: new Date(start),
                    end: new Date(end),
                  }
                : { type: 'relative', duration: parseDuration(DEFAULT_RANGE) }
            }
          />
        </SelectGroup>
        <form
          className="w-full max-w-[360px]"
          onSubmit={(e) => {
            e.preventDefault();
            if (trimmedSearch) {
              navigate({
                to: pathCreator.session({
                  sessionKey,
                  sessionId: trimmedSearch,
                }) as LinkComponentProps['to'],
              });
            }
          }}
        >
          <Search
            name="sessionId"
            placeholder="Search by session identifier"
            value={search}
            maxLength={512}
            autoFocus
            className="w-full"
            onUpdate={setSearch}
          />
        </form>
      </div>
      <div className="flex-1 overflow-y-auto">
        {error ? (
          <ErrorCard error={error} reset={() => refetch()} />
        ) : (
          <>
            <Table
              columns={columns}
              data={sessions}
              isLoading={isPending || isFetching}
              blankState={
                <TableBlankState
                  icon={<SessionsIcon />}
                  title={
                    trimmedSearch
                      ? `No session identifier found for "${trimmedSearch}"`
                      : `No sessions found for "${sessionKey}"`
                  }
                  actions={
                    <>
                      <Button appearance="outlined" label="Edit search" onClick={onEditSearch} />
                      <Button
                        label="Go to docs"
                        href={DOCS_URL}
                        target="_blank"
                        icon={<RiExternalLinkLine />}
                        iconSide="left"
                      />
                    </>
                  }
                />
              }
              onRowClick={(row) =>
                navigate({
                  to: pathCreator.session({
                    sessionKey: row.original.sessionKey,
                    sessionId: row.original.sessionId,
                  }) as LinkComponentProps['to'],
                })
              }
              getRowHref={(row) =>
                pathCreator.session({
                  sessionKey: row.original.sessionKey,
                  sessionId: row.original.sessionId,
                })
              }
            />
          </>
        )}
      </div>
    </div>
  );
}
