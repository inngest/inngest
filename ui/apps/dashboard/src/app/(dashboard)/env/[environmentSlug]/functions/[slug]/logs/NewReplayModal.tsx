import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Modal } from '@inngest/components/Modal';
import { IconReplay } from '@inngest/components/icons/Replay';
import * as Popover from '@radix-ui/react-popover';
import * as ToggleGroup from '@radix-ui/react-toggle-group';
import * as chrono from 'chrono-node';
import { useDebounce } from 'react-use';
import { toast } from 'sonner';

import { type TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import Input from '@/components/Forms/Input';
import { FunctionRunStatus } from '@/gql/graphql';

type NewReplayModalProps = {
  environmentSlug: string;
  functionSlug?: string;
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
  const [selectedStatuses, setSelectedStatuses] = useState<FunctionRunStatus[]>([
    FunctionRunStatus.Failed,
  ]);

  async function newReplay(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!timeRange) {
      toast.error('Please specify a valid time range.');
      return;
    }

    const replayFunctionMutation = async () => {
      // TODO: Implement this mutation
    };

    toast.promise(replayFunctionMutation, {
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
      <form onSubmit={newReplay}>
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
            <DateTimeRangeInput onChange={setTimeRange} />
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
            icon={<IconReplay className="h-5 w-5 text-white" />}
          />
        </div>
      </form>
    </Modal>
  );
}

type DateTimeRangeInputProps = {
  onChange: (timeRange: TimeRange) => void;
};

function DateTimeRangeInput({ onChange }: DateTimeRangeInputProps) {
  const [startDateTime, setStartDateTime] = useState<Date>();
  const [startDateTimeError, setStartDateTimeError] = useState<string>();
  const [endDateTime, setEndDateTime] = useState<Date>();
  const [endDateTimeError, setEndDateTimeError] = useState<string | undefined>();

  function onStartDateTimeChange(newStartDateTime: Date) {
    setStartDateTime(newStartDateTime);
    setStartDateTimeError(undefined);
    setEndDateTimeError(undefined);
    if (endDateTime && newStartDateTime > endDateTime) {
      setStartDateTimeError('Start time must be before end time.');
      return;
    }

    if (endDateTime) {
      onChange({ start: newStartDateTime, end: endDateTime });
    }
  }

  function onEndDateTimeChange(newEndDateTime: Date) {
    setEndDateTime(newEndDateTime);
    setStartDateTimeError(undefined);
    setEndDateTimeError(undefined);
    if (startDateTime && newEndDateTime < startDateTime) {
      setEndDateTimeError('End time must be after start time.');
      return;
    }

    if (startDateTime) {
      onChange({ start: startDateTime, end: newEndDateTime });
    }
  }

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <DateTimeInput onChange={onStartDateTimeChange} required />
        <span className="text-sm font-medium text-slate-400">to</span>
        <DateTimeInput onChange={onEndDateTimeChange} required />
      </div>
      {startDateTimeError && <p className="text-sm text-red-500">{startDateTimeError}</p>}
      {endDateTimeError && <p className="text-right text-sm text-red-500">{endDateTimeError}</p>}
    </div>
  );
}

type DateTimeInputProps = {
  onChange: (newDateTime: Date) => void;
  required?: boolean;
};

function DateTimeInput({ onChange, required }: DateTimeInputProps) {
  // TODO: This component's state management can probably be improved by using useReducer()
  const [inputString, setInputString] = useState<string>('');
  const [parsedDateTime, setParsedDateTime] = useState<Date>();
  useDebounce(
    () => {
      if (status === 'selected') return;
      const parsedDateTime = chrono.parseDate(inputString);
      if (!parsedDateTime) {
        setStatus('invalid');
        return;
      }
      setStatus('valid');
      setParsedDateTime(parsedDateTime);
    },
    350,
    [inputString]
  );
  const [status, setStatus] = useState<'typing' | 'invalid' | 'valid' | 'selected'>('invalid');

  function onInputChange(event: React.ChangeEvent<HTMLInputElement>) {
    setStatus('typing');
    setParsedDateTime(undefined);
    setInputString(event.target.value);
  }

  function applyDateTime(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.code === 'Enter' && parsedDateTime && inputString.length > 0) {
      event.preventDefault();
      onChange(parsedDateTime);
      setInputString(parsedDateTime.toLocaleString());
      setStatus('selected');
    }
  }

  const isPopoverOpen = status === 'valid';

  return (
    <Popover.Root open={isPopoverOpen}>
      <Popover.Anchor>
        <Input
          type="text"
          value={inputString}
          onChange={onInputChange}
          onKeyDown={applyDateTime}
          required={required}
        />
      </Popover.Anchor>
      <Popover.Portal>
        <Popover.Content
          className="shadow-floating z-[100] inline-flex items-center gap-2 space-y-4 rounded-md bg-white/95 p-2 text-sm text-slate-800 ring-1 ring-black/5 backdrop-blur-[3px]"
          sideOffset={5}
          onOpenAutoFocus={(event) => event.preventDefault()}
        >
          {parsedDateTime?.toLocaleString()}
          <kbd
            className="ml-auto flex h-6 w-6 items-center justify-center rounded bg-slate-100 p-2 font-sans text-xs"
            aria-label="Press Enter to apply the parsed date and time."
          >
            ↵
          </kbd>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
