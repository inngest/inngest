import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Modal } from '@inngest/components/Modal';
import { IconReplay } from '@inngest/components/icons/Replay';
import * as ToggleGroup from '@radix-ui/react-toggle-group';
import { toast } from 'sonner';
import { ulid } from 'ulid';
import { useMutation } from 'urql';

import { type TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import Input from '@/components/Forms/Input';
import { TimeRangeInput } from '@/components/TimeRangeInput';
import { graphql } from '@/gql';
import { FunctionRunStatus } from '@/gql/graphql';
import { useEnvironment } from '@/queries';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const GetFunctionEndedRunsCountDocument = graphql(`
  query GetFunctionEndedRunsCount(
    $environmentID: ID!
    $functionSlug: String!
    $timeRangeStart: Time!
    $timeRangeEnd: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        failedRuns: runsV2(
          filter: {
            status: [FAILED]
            lowerTime: $timeRangeStart
            upperTime: $timeRangeEnd
            timeField: STARTED_AT
          }
        ) {
          totalCount
        }
        canceledRuns: runsV2(
          filter: {
            status: [CANCELLED]
            lowerTime: $timeRangeStart
            upperTime: $timeRangeEnd
            timeField: STARTED_AT
          }
        ) {
          totalCount
        }
        succeededRuns: runsV2(
          filter: {
            status: [COMPLETED]
            lowerTime: $timeRangeStart
            upperTime: $timeRangeEnd
            timeField: STARTED_AT
          }
        ) {
          totalCount
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
    $statuses: [FunctionRunStatus!]
  ) {
    createFunctionReplay(
      input: {
        workspaceID: $environmentID
        workflowID: $functionID
        name: $name
        fromRange: $fromRange
        toRange: $toRange
        statuses: $statuses
      }
    ) {
      id
    }
  }
`);

type FunctionRunEndStatus =
  | FunctionRunStatus.Failed
  | FunctionRunStatus.Cancelled
  | FunctionRunStatus.Completed;

type NewReplayModalProps = {
  environmentSlug: string;
  functionSlug: string;
  isOpen: boolean;
  onClose: () => void;
};

export default function NewReplayModal({
  environmentSlug,
  functionSlug,
  isOpen,
  onClose,
}: NewReplayModalProps) {
  const router = useRouter();
  const [name, setName] = useState<string>('');
  const [timeRange, setTimeRange] = useState<TimeRange>();
  const [selectedStatuses, setSelectedStatuses] = useState<FunctionRunEndStatus[]>([
    FunctionRunStatus.Failed,
  ]);
  const [{ data: environment }] = useEnvironment({
    environmentSlug,
  });
  const { data, isLoading, error } = useGraphQLQuery({
    query: GetFunctionEndedRunsCountDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug,
      timeRangeStart: timeRange?.start ? timeRange.start.toISOString() : '',
      timeRangeEnd: timeRange?.end ? timeRange.end.toISOString() : '',
    },
    skip: !environment?.id || !timeRange?.start || !timeRange?.end,
  });
  const [{ fetching: isCreatingFunctionReplay }, createFunctionReplayMutation] = useMutation(
    CreateFunctionReplayDocument
  );

  const failedRunsCount = data?.environment?.function?.failedRuns?.totalCount ?? 0;
  const canceledRunsCount = data?.environment?.function?.canceledRuns?.totalCount ?? 0;
  const succeededRunsCount = data?.environment?.function?.succeededRuns?.totalCount ?? 0;

  const statusCounts: Record<FunctionRunEndStatus, number> = {
    [FunctionRunStatus.Failed]: failedRunsCount,
    [FunctionRunStatus.Cancelled]: canceledRunsCount,
    [FunctionRunStatus.Completed]: succeededRunsCount,
  } as const;

  const selectedRunsCount = selectedStatuses.reduce((acc, status) => acc + statusCounts[status], 0);

  async function createFunctionReplay(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!timeRange) {
      toast.error('Please specify a valid time range.');
      return;
    }

    if (selectedRunsCount === 0) {
      toast.error('No runs selected. Please specify a filter with at least one run.');
      return;
    }

    if (!environment?.id) {
      toast.error('Could not find environment. Please try again later.');
      return;
    }

    const functionID = data?.environment?.function?.id;

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
      success: (response) => {
        onClose();
        router.push(`/env/${environmentSlug}/functions/${encodeURIComponent(functionSlug)}/replay`);
        return 'Replay created!';
      },
      error: 'Could not replay function runs. Please try again later.',
    });
  }

  const statusOptions = [
    { label: 'Failed', value: FunctionRunStatus.Failed, count: failedRunsCount },
    { label: 'Canceled', value: FunctionRunStatus.Cancelled, count: canceledRunsCount },
    { label: 'Succeeded', value: FunctionRunStatus.Completed, count: succeededRunsCount },
  ];

  return (
    <Modal
      className="max-w-2xl p-0"
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
                value={name}
                minLength={3}
                maxLength={64}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </div>
          </div>
          <div className="flex justify-between gap-7 px-6 py-4">
            <div className="space-y-0.5">
              <span className="text-sm font-semibold text-slate-800">Time Range</span>
              <p className="text-xs text-slate-500">A time range to replay function runs from.</p>
            </div>
            <TimeRangeInput onChange={setTimeRange} />
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
            onValueChange={(selectedStatuses: FunctionRunEndStatus[]) => {
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
                  <FunctionRunStatusIcon status={value} className="mx-auto h-8" />
                  {label}
                </ToggleGroup.Item>
                <p aria-label={`Number of ${label} runs`} className="text-sm text-slate-500">
                  {count.toLocaleString(undefined, {
                    notation: 'compact',
                    compactDisplay: 'short',
                  })}{' '}
                  Runs
                </p>
              </div>
            ))}
          </ToggleGroup.Root>
        </div>
        <div className="flex flex-col gap-6 px-6 py-4">
          <div className="max-w-sm space-y-2 text-xs text-slate-500">
            <p>
              Replayed functions are re-run from the beginning. All previously run steps and
              function states will not be re-used during the replay.
            </p>
            <p>
              The <code>event.user</code> object will be empty for all runs in the replay.
            </p>
          </div>
          <div className="flex gap-2 self-end">
            <p className="inline-flex gap-1.5 text-slate-500">
              Total runs to be replayed:{' '}
              <span className="font-medium text-slate-800">
                {selectedRunsCount.toLocaleString(undefined, {
                  notation: 'compact',
                  compactDisplay: 'short',
                })}
              </span>
            </p>
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t border-slate-100 px-5 py-4">
          <Button type="button" appearance="outlined" label="Cancel" btnAction={onClose} />
          <Button
            label="Replay Function"
            kind="primary"
            type="submit"
            icon={<IconReplay className="h-5 w-5 text-white" />}
            disabled={isCreatingFunctionReplay}
          />
        </div>
      </form>
    </Modal>
  );
}
