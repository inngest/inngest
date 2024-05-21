import { isFunctionRunStatus, type FunctionRunStatus } from '@inngest/components/types/functionRun';

import { FunctionRunStatus as FunctionRunStatusEnum, RunsOrderByField } from '@/gql/graphql';

/**
 * Convert a run status union type into an enum. This is necessary because
 * TypeScript treats as enums as nominal types, which causes silly type errors.
 */
function toRunStatus(status: FunctionRunStatus): FunctionRunStatusEnum {
  switch (status) {
    case 'CANCELLED':
      return FunctionRunStatusEnum.Cancelled;
    case 'COMPLETED':
      return FunctionRunStatusEnum.Completed;
    case 'FAILED':
      return FunctionRunStatusEnum.Failed;
    case 'QUEUED':
      return FunctionRunStatusEnum.Queued;
    case 'RUNNING':
      return FunctionRunStatusEnum.Running;
  }
}

/**
 * Convert a run status string array into an enum array. Unrecognized statuses
 * are logged and will not be in the returned array.
 */
export function toRunStatuses(statuses: string[]): FunctionRunStatusEnum[] {
  const newValue: FunctionRunStatusEnum[] = [];

  for (const status of statuses) {
    if (isFunctionRunStatus(status)) {
      newValue.push(toRunStatus(status));
    } else {
      console.error(`unexpected status: ${status}`);
    }
  }

  return newValue;
}

/**
 * Convert a time field union type into an enum. This is necessary because
 * TypeScript treats as enums as nominal types, which causes silly type errors.
 */
export function toTimeField(time: string): RunsOrderByField | undefined {
  switch (time) {
    case 'ENDED_AT':
      return RunsOrderByField.EndedAt;
    case 'QUEUED_AT':
      return RunsOrderByField.QueuedAt;
    case 'STARTED_AT':
      return RunsOrderByField.StartedAt;
    default:
      console.error(`unexpected time field: ${time}`);
  }
}
