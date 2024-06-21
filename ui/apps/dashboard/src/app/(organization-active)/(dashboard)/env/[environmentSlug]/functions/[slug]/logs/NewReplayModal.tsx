'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { RangePicker } from '@inngest/components/DatePicker';
import { RunStatusIcon } from '@inngest/components/FunctionRunStatusIcons';
import { Link } from '@inngest/components/Link';
import { Modal } from '@inngest/components/Modal';
import { IconReplay } from '@inngest/components/icons/Replay';
import { subtractDuration } from '@inngest/components/utils/date';
import * as ToggleGroup from '@radix-ui/react-toggle-group';
import { toast } from 'sonner';
import { ulid } from 'ulid';
import { useMutation, useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import Input from '@/components/Forms/Input';
import Placeholder from '@/components/Placeholder';
import { graphql } from '@/gql';
import { ReplayRunStatus } from '@/gql/graphql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';

const GetBillingPlanDocument = graphql(`
  query GetBillingPlan {
    account {
      plan {
        id
        name
        features
      }
    }

    plans {
      name
      features
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
    query: GetBillingPlanDocument,
  });

  const logRetention = Number(planData?.account.plan?.features.log_retention);
  const upgradeCutoff = subtractDuration(new Date(), { days: logRetention || 7 });

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
      description="Select which function runs to replay."
      isOpen={isOpen}
      onClose={onClose}
    >
      <form className="divide-y divide-slate-100" onSubmit={createFunctionReplay}>
        <div className="divide-y divide-slate-100">
          <div className="flex items-start justify-between gap-7 px-6 py-4">
            <label htmlFor="replayName" className="block space-y-0.5">
              <span className="text-sm font-semibold text-slate-800">Replay Name</span>
              <p className="text-xs text-slate-500">Give your Replay a name to reference later.</p>
            </label>
            <div className="w-64">
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
          <div className="flex flex-row justify-between px-6 py-4">
            <div className="w-1/2 space-y-0.5">
              <span className="text-sm font-semibold text-slate-800">Date Range</span>
              <p className="text-xs text-slate-500">Select a specific range of function runs.</p>
            </div>
            <div className="w-1/2">
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
          <div className="space-y-0.5">
            <span className="text-sm font-semibold text-slate-800">Statuses</span>
            <p className="text-xs text-slate-500">Select the statuses you want to be replayed.</p>
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
              <div key={value} className="flex flex-1 flex-col items-center gap-3.5">
                <ToggleGroup.Item
                  className="flex w-full flex-col items-center gap-1 rounded-md bg-slate-100 py-6 text-sm font-semibold text-slate-800 hover:bg-slate-200 focus:outline-1 focus:outline-indigo-500 data-[state=on]:ring data-[state=on]:ring-indigo-500 data-[state=on]:ring-offset-2"
                  value={value}
                >
                  <RunStatusIcon status={value} className="mx-auto h-8" />
                  {label}
                </ToggleGroup.Item>
                {timeRange && (
                  <p aria-label={`Number of ${label} runs`} className="text-sm text-slate-500">
                    {isLoading ? (
                      <Placeholder className="top-px inline-flex h-3 w-3 bg-slate-200" />
                    ) : (
                      count.toLocaleString(undefined, {
                        notation: 'compact',
                        compactDisplay: 'short',
                      })
                    )}{' '}
                    Runs
                  </p>
                )}
              </div>
            ))}
          </ToggleGroup.Root>
        </div>
        <div className="flex flex-col gap-6 px-6 py-4">
          <div className="max-w-sm space-y-2 text-xs text-slate-500">
            <p>
              Replayed functions are re-run from the beginning. Previously run steps and function
              states will not be reused during the replay.
            </p>
            <p>
              The <code>event.user</code> object will be empty for all runs in the replay.
            </p>
          </div>
          {timeRange && (
            <div className="flex gap-2 self-end">
              <p className="inline-flex gap-1.5 text-slate-500">
                Total runs to be replayed:{' '}
                <span className="font-medium text-slate-800">
                  {isLoading ? (
                    <Placeholder className="top-px inline-flex h-4 w-4 bg-slate-200" />
                  ) : (
                    selectedRunsCount.toLocaleString(undefined, {
                      notation: 'compact',
                      compactDisplay: 'short',
                    })
                  )}
                </span>
              </p>
            </div>
          )}
        </div>
        <div className="flex justify-between border-t border-slate-100 px-5 py-4">
          <Link href="https://inngest.com/docs/platform/replay">Learn about Replay</Link>
          <div className="flex gap-2">
            <Button type="button" appearance="outlined" label="Cancel" btnAction={onClose} />
            <Button
              label="Replay Function"
              kind="primary"
              type="submit"
              icon={<IconReplay className="h-5 w-5 text-white" />}
              disabled={isCreatingFunctionReplay}
            />
          </div>
        </div>
      </form>
    </Modal>
  );
}
