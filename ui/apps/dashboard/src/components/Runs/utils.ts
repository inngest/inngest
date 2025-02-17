import type { Run } from '@inngest/components/RunsPage/types';
import { isFunctionRunStatus, type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { toMaybeDate } from '@inngest/components/utils/date';

import {
  FunctionRunStatus as FunctionRunStatusEnum,
  RunsOrderByField as FunctionRunTimeFieldEnum,
  type FunctionRunV2,
} from '@/gql/graphql';

/**
 * Convert a run status union type into an enum. This is necessary because
 * TypeScript treats as enums as nominal types, which causes silly type errors.
 */
export function toRunStatus(status: FunctionRunStatus): FunctionRunStatusEnum {
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
    case 'PAUSED':
      return FunctionRunStatusEnum.Paused;
    case 'SKIPPED':
      return FunctionRunStatusEnum.Skipped;
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
export function toTimeField(time: string): FunctionRunTimeFieldEnum | undefined {
  switch (time) {
    case 'ENDED_AT':
      return FunctionRunTimeFieldEnum.EndedAt;
    case 'QUEUED_AT':
      return FunctionRunTimeFieldEnum.QueuedAt;
    case 'STARTED_AT':
      return FunctionRunTimeFieldEnum.StartedAt;
    default:
      console.error(`unexpected time field: ${time}`);
  }
}

type PickedFunctionRunV2 = Pick<
  FunctionRunV2,
  'id' | 'queuedAt' | 'startedAt' | 'status' | 'endedAt' | 'eventName' | 'isBatch' | 'cronSchedule'
>;
type PickedFunctionRunV2EdgeWithNode = {
  node: PickedFunctionRunV2 & {
    app: {
      externalID: string;
      name: string;
    };
    function: {
      name: string;
      slug: string;
    };
  };
};

/**
 * Parses the runs data into the table format
 */
export function parseRunsData(runsData: PickedFunctionRunV2EdgeWithNode[] | undefined): Run[] {
  return (
    runsData?.map((edge) => {
      const startedAt = toMaybeDate(edge.node.startedAt);
      let durationMS = null;
      if (startedAt) {
        durationMS = (toMaybeDate(edge.node.endedAt) ?? new Date()).getTime() - startedAt.getTime();
      }

      return {
        ...edge.node,
        durationMS,
      };
    }) ?? []
  );
}
