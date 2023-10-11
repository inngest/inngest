import { TRIGGER_TYPE } from '@/components/Pill/TriggerPill';
import type { GetFunctionsQuery, GetFunctionsUsageQuery } from '@/gql/graphql';
import type { FunctionTableRow } from './FunctionTable';

export type FunctionList = {
  hasNextPage: boolean;
  isLoading: boolean;
  latestLoadedPage: number;
  latestRequestedPage: number;
  rows: FunctionTableRow[];
};

/**
 * Transform GraphQL data and append it to the function list.
 */
export function appendFunctionList(
  functionList: FunctionList,
  functions: GetFunctionsQuery['workspace']['workflows']
): FunctionList {
  const rows = functions.data.map((fn) => {
    const triggers = (fn.current?.triggers ?? []).map((trigger) => {
      const triggerType = trigger.eventName ? TRIGGER_TYPE.event : TRIGGER_TYPE.schedule;

      const value =
        (triggerType === TRIGGER_TYPE.event ? trigger.eventName : trigger.schedule) ?? 'unknown';

      return {
        type: triggerType,
        value,
      };
    });

    return {
      name: fn.name,
      slug: fn.slug,
      isArchived: fn.isArchived,
      isActive: Boolean(fn.current),
      triggers,
      failureRate: undefined,
      usage: undefined,
    };
  });

  return {
    ...functionList,
    rows: [...functionList.rows, ...rows],
  };
}

/**
 * Transform GraphQL data and update the function list.
 */
export function updateFunctionListWithUsage(
  functionList: FunctionList,
  functionUsages: GetFunctionsUsageQuery['workspace']['workflows']['data']
): FunctionList {
  let rows = [...functionList.rows];

  functionUsages.forEach((fnUsage) => {
    const dailyStartCount = fnUsage.dailyStarts.total;
    const dailyFailureCount = fnUsage.dailyFailures.total;

    // Calculates the daily failure rate percentage and rounds it up to 2 decimal places
    const failureRate =
      dailyStartCount === 0 ? 0 : Math.round((dailyFailureCount / dailyStartCount) * 10000) / 100;

    // Creates an array of objects containing the start and failure count for each usage slot (1 hour)
    const slots = fnUsage.dailyStarts.data.map((usageSlot, index) => ({
      startCount: usageSlot.count,
      failureCount: fnUsage.dailyFailures.data[index]?.count ?? 0,
    }));

    const usage = {
      slots,
      total: dailyStartCount,
    };

    const rowIndex = rows.findIndex((row) => row.slug === fnUsage.slug);
    const row = rows[rowIndex];
    if (!row) {
      return;
    }

    rows[rowIndex] = {
      ...row,
      failureRate: failureRate,
      usage,
    };
  });

  return {
    ...functionList,
    rows,
  };
}
