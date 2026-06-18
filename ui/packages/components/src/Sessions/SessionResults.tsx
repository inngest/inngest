import { useCallback, useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
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

type SessionResultsProps = {
  envID: string;
  // Max selectable range in days, from the account's history entitlement.
  maxRangeDays: number;
  sessionKey: string;
  last?: string;
  start?: string;
  end?: string;
  pathCreator: {
    session: (params: { sessionKey: string; sessionId: string }) => LinkComponentProps['to'];
    function: (params: { functionSlug: string }) => LinkComponentProps['to'];
  };
  getSessions: (params: {
    sessionKey: string;
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
    queryKey: ['sessions', envID, sessionKey, startTime, endTime],
    queryFn: () => getSessions({ sessionKey, startTime, endTime }),
    refetchOnWindowFocus: false,
  });

  const sessions = useMemo(() => data ?? [], [data]);

  return (
    <div className="bg-canvasBase text-basis flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="flex flex-col gap-4 px-3 pb-3 pt-6">
        <h1 className="text-basis text-xl font-medium">
          <span className="font-mono">{sessionKey}</span> Results
        </h1>
        <TimeFilter
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
                  title={`No sessions found for "${sessionKey}"`}
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
                  }),
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
