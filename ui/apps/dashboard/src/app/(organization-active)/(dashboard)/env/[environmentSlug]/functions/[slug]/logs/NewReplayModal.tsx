'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import { RangePicker } from '@inngest/components/DatePicker';
import { Input } from '@inngest/components/Forms/Input';
import { RunStatusIcon } from '@inngest/components/FunctionRunStatusIcons';
import { Link } from '@inngest/components/Link';
import { Modal } from '@inngest/components/Modal';
import { IconReplay } from '@inngest/components/icons/Replay';
import { subtractDuration } from '@inngest/components/utils/date';
import * as ToggleGroup from '@radix-ui/react-toggle-group';
import { RiInformationLine } from '@remixicon/react';
import { toast } from 'sonner';
import { ulid } from 'ulid';
import { useMutation, useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { ReplayRunStatus } from '@/gql/graphql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

const GetAccountEntitlementsDocument = graphql(`
  query GetAccountEntitlements {
    account {
      entitlements {
        history {
          limit
        }
      }
    }
  }
`);

const GetReplayRunCountsDocument = graphql(`
  query GetReplayRunCounts($environmentID: ID!, $functionSlug: String!, $from: Time!, $to: Time!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        replayCounts: replayCounts(from: $from, to: $to) {
          completedCount
          failedCount
          cancelledCount
          skippedPausedCount
        }
      }
    }
  }
`);

const CreateFunctionReplayDocument = graphql(`
  mutation CreateFunctionReplay(
    $environmentID: UUID!
    $functionID: UUID!
    $name: String!
    $fromRange: ULID!
    $toRange: ULID!
    $statuses: [ReplayRunStatus!]
  ) {
    createFunctionReplay(
      input: {
        workspaceID: $environmentID
        workflowID: $functionID
        name: $name
        fromRange: $fromRange
        toRange: $toRange
        statusesV2: $statuses
      }
    ) {
      id
    }
  }
`);

type SelectableStatuses =
  | ReplayRunStatus.Failed
  | ReplayRunStatus.Cancelled
  | ReplayRunStatus.Completed
  | ReplayRunStatus.SkippedPaused;

type NewReplayModalProps = {
  functionSlug: string;
  isOpen: boolean;
  onClose: () => void;
};

export type DateRange = {
  start?: Date;
  end?: Date;
  key?: string;
};

export default function NewReplayModal({ functionSlug, isOpen, onClose }: NewReplayModalProps) {
  const router = useRouter();
  const [name, setName] = useState<string>('');
  const [timeRange, setTimeRange] = useState<DateRange>();
  const [selectedStatuses, setSelectedStatuses] = useState<SelectableStatuses[]>([
    ReplayRunStatus.Failed,
  ]);
  const environment = useEnvironment();

  const [{ data: planData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });

  const logRetention = planData?.account.entitlements.history.limit || 7;
  const upgradeCutoff = subtractDuration(new Date(), { days: logRetention });

  const { data, isLoading } = useSkippableGraphQLQuery({
    query: GetReplayRunCountsDocument,
    variables: {
      environmentID: environment.id,
      functionSlug,
      from: timeRange?.start ? timeRange.start.toISOString() : '',
      to: timeRange?.end ? timeRange.end.toISOString() : '',
    },
    skip: !timeRange || !timeRange.start || !timeRange.end,
  });
  const [{ fetching: isCreatingFunctionReplay }, createFunctionReplayMutation] = useMutation(
    CreateFunctionReplayDocument
  );

  const failedRunsCount = data?.environment.function?.replayCounts.failedCount ?? 0;
  const cancelledRunsCount = data?.environment.function?.replayCounts.cancelledCount ?? 0;
  const succeededRunsCount = data?.environment.function?.replayCounts.completedCount ?? 0;
  const pausedRunsCount = data?.environment.function?.replayCounts.skippedPausedCount ?? 0;

  const statusCounts: Record<SelectableStatuses, number> = {
    [ReplayRunStatus.Failed]: failedRunsCount,
    [ReplayRunStatus.Cancelled]: cancelledRunsCount,
    [ReplayRunStatus.Completed]: succeededRunsCount,
    [ReplayRunStatus.SkippedPaused]: pausedRunsCount,
  } as const;

  const selectedRunsCount = selectedStatuses.reduce((acc, status) => acc + statusCounts[status], 0);

  async function createFunctionReplay(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!timeRange?.start || !timeRange.end) {
      toast.error('Please specify a start and end date.');
      return;
    }

    if (selectedRunsCount === 0) {
      toast.error('No runs selected. Please specify a filter with at least one run.');
      return;
    }

    const functionID = data?.environment.function?.id;

    if (!functionID) {
      toast.error('Could not find function. Please try again later.');
      return;
    }

    const createFunctionReplayPromise = createFunctionReplayMutation({
      environmentID: environment.id,
      functionID: functionID,
      name,
      fromRange: ulid(timeRange.start.valueOf()),
      toRange: ulid(timeRange.end.valueOf()),
      statuses: selectedStatuses,
    });

    toast.promise(createFunctionReplayPromise, {
      loading: 'Loading...',
      success: () => {
        onClose();
        router.push(
          `/env/${environment.slug}/functions/${encodeURIComponent(functionSlug)}/replay`
        );
        return 'Replay created!';
      },
      error: 'Could not replay function runs. Please try again later.',
    });
  }

  const statusOptions = [
    { label: 'Failed', value: ReplayRunStatus.Failed, count: failedRunsCount },
    { label: 'Canceled', value: ReplayRunStatus.Cancelled, count: cancelledRunsCount },
    { label: 'Succeeded', value: ReplayRunStatus.Completed, count: succeededRunsCount },
    { label: 'Skipped', value: ReplayRunStatus.SkippedPaused, count: pausedRunsCount },
  ];

  return (
    <Modal
      className="max-w-3xl p-0"
      title={
        <span className="inline-flex items-center gap-1">
          <IconReplay className="h-6 w-6" />
          Replay Function
        </span>
      }
      isOpen={isOpen}
      onClose={onClose}
    >
      <form onSubmit={createFunctionReplay}>
        <div>
          <div className="flex flex-col items-start justify-between gap-2 px-6 py-4">
            <label htmlFor="replayName">
              <span className="text-basis text-sm font-semibold">Replay Name</span>
              <p className="text-muted text-sm">
                Provide a unique name for this replay group to easily identify the runs being
                replayed.
              </p>
            </label>
            <div className="w-full">
              <Input
                type="text"
                id="replayName"
                placeholder="Incident #1234"
                value={name}
                minLength={3}
                maxLength={64}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </div>
          </div>
          <div className="flex flex-col justify-between gap-2 px-6 py-4">
            <div>
              <span className="text-basis text-sm font-semibold">Date Range</span>
              <p className="text-muted text-sm">
                Choose the time range for when the runs were queued.
              </p>
            </div>
            <div className="w-full">
              <RangePicker
                upgradeCutoff={upgradeCutoff}
                onChange={(range) =>
                  setTimeRange(
                    range.type === 'relative'
                      ? { start: subtractDuration(new Date(), range.duration), end: new Date() }
                      : { start: range.start, end: range.end }
                  )
                }
                className="w-full"
              />
            </div>
          </div>
        </div>
        <div className="space-y-5 px-6 py-4">
          <div>
            <span className="text-basis text-sm font-semibold">Statuses</span>
            <p className="text-muted text-sm">Select the statuses you wish to replay.</p>
          </div>
          <ToggleGroup.Root
            type="multiple"
            value={selectedStatuses}
            onValueChange={(selectedStatuses: SelectableStatuses[]) => {
              if (selectedStatuses.length === 0) return; // Must have at least one status selected
              setSelectedStatuses(selectedStatuses);
            }}
            className="flex gap-5"
          >
            {statusOptions.map(({ label, value, count }) => (
              <div key={value} className="flex flex-1">
                <ToggleGroup.Item
                  className="focus:ring-primary-moderate data-[state=on]:bg-success text-basis border-subtle hover:bg-canvasSubtle data-[state=on]:border-primary-moderate items-left flex w-full flex-col gap-1 rounded-md border p-3 text-sm"
                  value={value}
                >
                  <RunStatusIcon status={value} className="mx-auto h-8" />
                  {label}
                  {!timeRange && <p className="text-muted text-sm">-- runs</p>}
                  {timeRange && (
                    <p aria-label={`Number of ${label} runs`} className="text-muted text-sm">
                      {isLoading ? (
                        <span>Loading</span>
                      ) : (
                        count.toLocaleString(undefined, {
                          notation: 'compact',
                          compactDisplay: 'short',
                        })
                      )}{' '}
                      runs {isLoading ? '...' : undefined}
                    </p>
                  )}
                </ToggleGroup.Item>
              </div>
            ))}
          </ToggleGroup.Root>
        </div>
        <div className="px-6 py-4">
          <div className="text-muted bg-canvasSubtle rounded-md px-6 py-4 text-sm">
            <p>
              Note: Replayed functions are re-run from the beginning. Previously run steps and
              function states will not be reused during the replay. The{' '}
              <InlineCode>event.user</InlineCode> object will be empty for all runs in the replay.
            </p>
            <Link target="_blank" href="https://inngest.com/docs/platform/replay">
              Learn more about replay
            </Link>
          </div>
        </div>
        <div className="border-subtle flex items-center justify-between gap-2 border-t px-5 py-4">
          {!timeRange && <p></p>}
          {timeRange && !isLoading && (
            <div className="flex items-center gap-2">
              <p className="text-muted inline-flex items-center gap-1.5 text-sm">
                <RiInformationLine className="h-5 w-5" />A total of{' '}
                <span className="font-bold">
                  {selectedRunsCount.toLocaleString(undefined, {
                    notation: 'compact',
                    compactDisplay: 'short',
                  })}
                </span>
                runs will be replayed.
              </p>
            </div>
          )}
          <div className="flex gap-2">
            <Button
              type="button"
              appearance="outlined"
              kind="secondary"
              label="Cancel"
              onClick={onClose}
            />
            <Button
              label="Replay Function"
              kind="primary"
              type="submit"
              disabled={isCreatingFunctionReplay}
            />
          </div>
        </div>
      </form>
    </Modal>
  );
}
