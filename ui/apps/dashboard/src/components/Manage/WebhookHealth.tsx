import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Skeleton } from '@inngest/components/Skeleton';
import { Table } from '@inngest/components/Table';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { formatLocalTime, formatUTCTime } from '@inngest/components/utils/time';
import { RiArrowRightSLine, RiInformationLine } from '@remixicon/react';
import type { ColumnDef } from '@tanstack/react-table';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { graphql } from '@/gql';
import { WebhookFailureReason, type WebhookHealthQuery } from '@/gql/graphql';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import {
  useGraphQLQuery,
  useSkippableGraphQLQuery,
} from '@/utils/useGraphQLQuery';

const WebhookHealthDocument = graphql(`
  query WebhookHealth($environmentID: ID!, $keyID: ID!, $from: Time!) {
    environment: workspace(id: $environmentID) {
      ingestKey(id: $keyID) {
        eventCount(from: $from)
        failureGroups(from: $from) {
          reason
          count
          firstSeen
          lastSeen
          message
          requestID
        }
      }
    }
  }
`);

const WebhookFailuresDocument = graphql(`
  query WebhookFailures(
    $environmentID: ID!
    $keyID: ID!
    $from: Time!
    $reason: WebhookFailureReason!
    $after: String
  ) {
    environment: workspace(id: $environmentID) {
      ingestKey(id: $keyID) {
        failures(first: 20, after: $after, from: $from, reasons: [$reason]) {
          data {
            receivedAt
            count
            requestID
            message
            eventsReceived
            eventsIngested
          }
          pageInfo {
            hasNextPage
            endCursor
          }
        }
      }
    }
  }
`);

type FailureGroup =
  WebhookHealthQuery['environment']['ingestKey']['failureGroups'][number];

type GroupRow = FailureGroup & { id: string };

const reasonLabels: Record<WebhookFailureReason, string> = {
  [WebhookFailureReason.BodyTooLarge]: 'Request body too large',
  [WebhookFailureReason.EnvNotFound]: 'Environment not found',
  [WebhookFailureReason.Filtered]: 'Filtered',
  [WebhookFailureReason.InternalError]: 'Internal error',
  [WebhookFailureReason.InvalidBody]: 'Invalid request body',
  [WebhookFailureReason.InvalidEvent]: 'Invalid event',
  [WebhookFailureReason.PublishError]: 'Failed to publish event',
  [WebhookFailureReason.ResponseError]: 'Failed to send response',
  [WebhookFailureReason.TooManyEvents]: 'Too many events',
  [WebhookFailureReason.TransformError]: 'Transform failed',
  [WebhookFailureReason.TransformInvalidOutput]:
    'Transform returned invalid output',
};

function reasonLabel(reason: WebhookFailureReason): string {
  return reasonLabels[reason];
}

function shortAge(date: string): string {
  const minutes = Math.floor((Date.now() - new Date(date).getTime()) / 60_000);
  if (minutes < 1) return 'now';
  if (minutes < 60) return `${minutes}m`;
  if (minutes < 24 * 60) return `${Math.floor(minutes / 60)}h`;
  return `${Math.floor(minutes / (24 * 60))}d`;
}

function shortMinute(date: Date): string {
  const time = date.toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  });
  if (date.toDateString() === new Date().toDateString()) return time;
  const day = date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
  });
  return `${day} ${time}`;
}

function plural(count: number, word: string): string {
  return `${count.toLocaleString()} ${word}${count === 1 ? '' : 's'}`;
}

function TimeCell({
  value,
  absolute = false,
}: {
  value: string;
  absolute?: boolean;
}) {
  const date = new Date(value);
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="text-muted whitespace-nowrap text-sm">
          {absolute ? shortMinute(date) : shortAge(value)}
        </span>
      </TooltipTrigger>
      <TooltipContent className="whitespace-nowrap">
        <div>{formatLocalTime(date)}</div>
        <div className="text-light text-xs">{formatUTCTime(date)} UTC</div>
        {absolute && (
          <div className="text-light text-xs">
            {shortAge(value) === 'now' ? 'Just now' : `${shortAge(value)} ago`}
          </div>
        )}
      </TooltipContent>
    </Tooltip>
  );
}

function MessageCell({
  message,
  requestID,
  className,
}: {
  message: string;
  requestID: string;
  className?: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn('block truncate text-sm', className)}>
          {message}
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-lg whitespace-pre-wrap break-words">
        {message}
        {requestID && (
          <div className="text-light mt-1 text-xs">Request ID: {requestID}</div>
        )}
      </TooltipContent>
    </Tooltip>
  );
}

const getColumns = (expandedIDs: string[]): ColumnDef<GroupRow>[] => [
  {
    id: 'expand',
    header: '',
    size: 32,
    cell: ({ row }) => (
      <RiArrowRightSLine
        className={cn(
          'text-muted h-4 w-4 transition-transform',
          expandedIDs.includes(row.original.id) && 'rotate-90',
        )}
      />
    ),
  },
  {
    accessorKey: 'reason',
    header: 'Reason',
    cell: ({ row }) => (
      <span className="whitespace-nowrap text-sm">
        {reasonLabel(row.original.reason)}
      </span>
    ),
  },
  {
    accessorKey: 'message',
    header: 'Message (latest)',
    cell: ({ row }) => (
      <MessageCell
        message={row.original.message}
        requestID={row.original.requestID}
        className="max-w-md"
      />
    ),
  },
  {
    accessorKey: 'count',
    header: () => <span className="block w-full text-right">Count</span>,
    cell: ({ row }) => (
      <span className="block text-right text-sm tabular-nums">
        {row.original.count.toLocaleString()}
      </span>
    ),
  },
  {
    accessorKey: 'lastSeen',
    header: 'Last seen',
    cell: ({ row }) => <TimeCell value={row.original.lastSeen} />,
  },
  {
    accessorKey: 'firstSeen',
    header: 'First seen',
    cell: ({ row }) => <TimeCell value={row.original.firstSeen} />,
  },
];

const failureGridClassName =
  'grid grid-cols-[6.5rem_4rem_minmax(0,1fr)_minmax(0,18rem)] gap-4';

function FailuresPage({
  keyID,
  from,
  reason,
  after,
  isFirst,
  onLoadMore,
}: {
  keyID: string;
  from: string;
  reason: WebhookFailureReason;
  after: string | null;
  isFirst: boolean;
  onLoadMore?: (cursor: string) => void;
}) {
  const environment = useEnvironment();
  const res = useGraphQLQuery({
    query: WebhookFailuresDocument,
    variables: { environmentID: environment.id, keyID, from, reason, after },
  });

  if (!res.data) {
    if (res.error) {
      return (
        <p className="text-error py-2 text-sm">
          Failed to load failures: {res.error.message}
        </p>
      );
    }
    return (
      <div className="flex flex-col gap-2 py-2">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-2/3" />
      </div>
    );
  }

  const { data, pageInfo } = res.data.environment.ingestKey.failures;
  const nextCursor = pageInfo.endCursor;

  return (
    <>
      {isFirst && data.length === 0 && (
        <p className="text-subtle py-2 text-sm">No matching failures</p>
      )}
      {data.map((failure, i) => {
        const message =
          failure.eventsIngested < failure.eventsReceived
            ? `${failure.message} (${failure.eventsIngested} of ${failure.eventsReceived} ingested)`
            : failure.message;
        return (
          <div
            key={`${failure.requestID}-${i}`}
            className={cn(failureGridClassName, 'items-center py-1.5')}
          >
            <TimeCell value={failure.receivedAt} absolute />
            <span className="text-right tabular-nums">
              {failure.count.toLocaleString()}
            </span>
            <MessageCell message={message} requestID={failure.requestID} />
            <span className="text-muted truncate font-mono text-xs">
              {failure.requestID}
            </span>
          </div>
        );
      })}
      {onLoadMore && pageInfo.hasNextPage && nextCursor && (
        <div className="pt-2">
          <Button
            kind="secondary"
            appearance="outlined"
            size="small"
            label="Load more"
            onClick={() => onLoadMore(nextCursor)}
          />
        </div>
      )}
    </>
  );
}

function GroupFailures({
  keyID,
  from,
  reason,
}: {
  keyID: string;
  from: string;
  reason: WebhookFailureReason;
}) {
  const [cursors, setCursors] = useState<(string | null)[]>([null]);

  return (
    <div className="bg-canvasSubtle px-4 py-3">
      <div
        className={cn(
          failureGridClassName,
          'text-muted border-subtle border-b pb-1 text-xs font-medium',
        )}
      >
        <span>Minute</span>
        <span className="text-right">Count</span>
        <span>Latest message</span>
        <span>Latest request ID</span>
      </div>
      {cursors.map((after, i) => (
        <FailuresPage
          key={after ?? ''}
          keyID={keyID}
          from={from}
          reason={reason}
          after={after}
          isFirst={i === 0}
          onLoadMore={
            i === cursors.length - 1
              ? (cursor) => setCursors((prev) => [...prev, cursor])
              : undefined
          }
        />
      ))}
    </div>
  );
}

function ReasonChip({
  label,
  selected,
  neutral,
  onClick,
}: {
  label: string;
  selected: boolean;
  neutral: boolean;
  onClick: () => void;
}) {
  if (neutral) {
    return (
      <Button
        kind="secondary"
        appearance="outlined"
        size="small"
        label={label}
        onClick={onClick}
        className={cn(selected && 'border-contrast bg-canvasMuted')}
      />
    );
  }
  return (
    <Button
      kind="danger"
      appearance={selected ? 'solid' : 'outlined'}
      size="small"
      label={label}
      onClick={onClick}
    />
  );
}

export default function WebhookHealth({ keyID }: { keyID: string }) {
  const environment = useEnvironment();
  const flag = useBooleanFlag('webhook-failure-rows');
  const enabled = flag.isReady && flag.value;
  const features = useAccountFeatures();
  const history = features.data?.history ?? 7;
  const days = history > 0 ? Math.min(7, history) : 7;
  const span = days === 1 ? 'day' : `${days} days`;
  const [now] = useState(() => Date.now());
  const from = useMemo(
    () => new Date(now - days * 24 * 60 * 60 * 1000).toISOString(),
    [now, days],
  );
  const [reason, setReason] = useState<WebhookFailureReason | null>(null);
  const [expandedIDs, setExpandedIDs] = useState<string[]>([]);

  const res = useSkippableGraphQLQuery({
    query: WebhookHealthDocument,
    skip: !enabled || (!features.data && !features.error),
    variables: { environmentID: environment.id, keyID, from },
  });

  const groups = useMemo<GroupRow[]>(
    () =>
      (res.data?.environment.ingestKey.failureGroups ?? []).map((g) => ({
        ...g,
        id: g.reason,
      })),
    [res.data],
  );
  const columns = useMemo(() => getColumns(expandedIDs), [expandedIDs]);

  if (!enabled || res.isSkipped) {
    return null;
  }

  const heading = (
    <h2 className="pb-1 text-lg font-semibold">Recent activity</h2>
  );

  if (!res.data) {
    return (
      <div className="mt-8 pt-3">
        {heading}
        {res.error ? (
          <p className="text-error text-sm">
            Failed to load recent activity: {res.error.message}
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            <Skeleton className="h-5 w-80" />
            <Skeleton className="h-7 w-96" />
            <Skeleton className="h-40 w-full" />
          </div>
        )}
      </div>
    );
  }

  const events = res.data.environment.ingestKey.eventCount;
  let failed = 0;
  let filtered = 0;
  for (const g of groups) {
    if (g.reason === WebhookFailureReason.Filtered) {
      filtered += g.count;
    } else {
      failed += g.count;
    }
  }
  const chips = [...groups].sort((a, b) => b.count - a.count);
  const rows = reason ? chips.filter((g) => g.reason === reason) : chips;

  return (
    <div className="mt-8 pt-3">
      {heading}
      {res.error && (
        <p className="text-error mb-2 text-sm">
          Failed to refresh recent activity: {res.error.message}
        </p>
      )}
      {events === 0 && groups.length === 0 ? (
        <p className="text-subtle text-sm">
          No requests received in the last {span}.
        </p>
      ) : (
        <>
          <div className="text-subtle mb-6 text-sm">
            <p className="flex items-center gap-1">
              Last {span}: {plural(events, 'event')} ingested
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-flex">
                    <RiInformationLine className="text-muted h-4 w-4" />
                  </span>
                </TooltipTrigger>
                <TooltipContent className="max-w-xs">
                  Counts events, not requests. A request that only partly
                  ingested also shows up as a failure.
                </TooltipContent>
              </Tooltip>
            </p>
            <p>
              {plural(failed, 'failed request')}
              {filtered > 0 && <> · {plural(filtered, 'filtered request')}</>}
            </p>
          </div>
          {groups.length > 0 && (
            <>
              <div className="mb-4 flex flex-wrap gap-2">
                <ReasonChip
                  label="All"
                  neutral
                  selected={reason === null}
                  onClick={() => setReason(null)}
                />
                {chips.map((g) => (
                  <ReasonChip
                    key={g.reason}
                    label={`${reasonLabel(
                      g.reason,
                    )} (${g.count.toLocaleString()})`}
                    neutral={g.reason === WebhookFailureReason.Filtered}
                    selected={reason === g.reason}
                    onClick={() => setReason(g.reason)}
                  />
                ))}
              </div>
              <div className="border-subtle overflow-x-auto rounded-md border">
                <Table
                  columns={columns}
                  data={rows}
                  headerStyle="subtle"
                  expandedIDs={expandedIDs}
                  onRowClick={(row) =>
                    setExpandedIDs((prev) =>
                      prev.includes(row.original.id)
                        ? prev.filter((id) => id !== row.original.id)
                        : [...prev, row.original.id],
                    )
                  }
                  renderSubComponent={({ row }) => (
                    <GroupFailures
                      key={row.original.reason}
                      keyID={keyID}
                      from={from}
                      reason={row.original.reason}
                    />
                  )}
                  blankState={
                    <p className="text-subtle p-4 text-sm">
                      {reason ? 'No matching requests' : 'No failures'}
                    </p>
                  }
                />
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}
