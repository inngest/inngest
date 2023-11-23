import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Modal } from '@inngest/components/Modal';
import { IconReplay } from '@inngest/components/icons/Replay';
import * as ToggleGroup from '@radix-ui/react-toggle-group';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { type TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import Input from '@/components/Forms/Input';
import { TimeRangeInput } from '@/components/TimeRangeInput';
import { graphql } from '@/gql';
import { FunctionRunStatus } from '@/gql/graphql';

const CreateFunctionReplayDocument = graphql(`
  mutation CreateFunctionReplay(
    $environmentID: UUID!
    $functionID: UUID!
    $name: String!
    $fromRange: Time!
    $toRange: Time!
    $statuses: [RunStatus!]
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

type NewReplayModalProps = {
  environmentID: string;
  functionID: string;
  isOpen: boolean;
  onClose: () => void;
};

export default function NewReplayModal({
  environmentID,
  functionID,
  isOpen,
  onClose,
}: NewReplayModalProps) {
  const router = useRouter();
  const [name, setName] = useState<string>('');
  const [timeRange, setTimeRange] = useState<TimeRange>();
  const [selectedStatuses, setSelectedStatuses] = useState<FunctionRunStatus[]>([
    FunctionRunStatus.Failed,
  ]);
  const [{ fetching: isCreatingFunctionReplay }, createFunctionReplayMutation] = useMutation(
    CreateFunctionReplayDocument
  );

  async function createFunctionReplay(event: React.FormEvent<HTMLFormElement>) {
    console.log('newReplay');
    event.preventDefault();
    if (!timeRange) {
      toast.error('Please specify a valid time range.');
      return;
    }

    console.log('timeRange', timeRange);
    console.log('name', name);

    const createFunctionReplayPromise = createFunctionReplayMutation({
      environmentID: environmentID,
      functionID: functionID,
      name,
      fromRange: timeRange.start.toISOString(),
      toRange: timeRange.end.toISOString(),
      statuses: selectedStatuses,
    });

    toast.promise(createFunctionReplayPromise, {
      loading: 'Loading...',
      success: () => {
        router.refresh();
        onClose();
        return 'Replay created!';
      },
      error: 'Could not replay function runs. Please try again later.',
    });
  }

  const statusOptions = [
    { label: 'Failed', value: FunctionRunStatus.Failed },
    { label: 'Canceled', value: FunctionRunStatus.Cancelled },
    { label: 'Completed', value: FunctionRunStatus.Completed },
  ];

  return (
    <Modal
      className="max-w-6xl p-0"
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
      <form onSubmit={createFunctionReplay}>
        <div className="divide-y divide-gray-900/10 border-b border-slate-100">
          <div className="flex items-start justify-between gap-7 px-6 py-4">
            <label htmlFor="name" className="block space-y-0.5">
              <span className="text-sm font-semibold text-slate-800">Replay Name</span>
              <p className="text-xs font-medium text-slate-500">
                Give your Replay a name to reference later.
              </p>
            </label>
            <div className="w-64">
              <Input
                type="text"
                id="name"
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
              <p className="text-xs font-medium text-slate-500">
                Specify a time range to replay function runs from.
              </p>
            </div>
            <TimeRangeInput onChange={setTimeRange} />
          </div>
        </div>
        <div className="space-y-5 px-6 py-4">
          <div className="space-y-0.5">
            <span className="text-sm font-semibold text-slate-800">Statuses</span>
            <p className="text-xs font-medium text-slate-500">
              Select the statuses you want to be replayed.
            </p>
          </div>
          <ToggleGroup.Root
            type="multiple"
            value={selectedStatuses}
            onValueChange={(selectedStatuses: FunctionRunStatus[]) => {
              if (selectedStatuses.length === 0) return; // Must have at least one status selected
              setSelectedStatuses(selectedStatuses);
            }}
            className="flex gap-5"
          >
            {statusOptions.map(({ label, value }) => (
              <ToggleGroup.Item
                key={value}
                className="flex flex-1 flex-col items-center gap-1 rounded-md bg-slate-100 py-6 text-sm font-semibold text-slate-800 hover:bg-slate-200 focus:outline-1 focus:outline-indigo-500 data-[state=on]:ring-2 data-[state=on]:ring-indigo-500 data-[state=on]:ring-offset-2"
                value={value}
              >
                <FunctionRunStatusIcon status={value} className="mx-auto h-8" />
                {label}
              </ToggleGroup.Item>
            ))}
          </ToggleGroup.Root>
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
