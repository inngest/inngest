import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import { GetHistoryItemOutputDocument } from '@/store/generated';

export function useGetHistoryItemOutput(runID: string | null) {
  return useCallback(
    (historyItemID: string) => {
      if (!runID) {
        // Should be unreachable.
        return new Promise<string>((resolve) => resolve(''));
      }

      return getHistoryItemOutput({ historyItemID, runID });
    },
    [runID]
  );
}

async function getHistoryItemOutput({
  historyItemID,
  runID,
}: {
  historyItemID: string;
  runID: string;
}): Promise<string | undefined> {
  // TODO: How to get type annotations? It returns `any`.
  const res: unknown = await client.request(GetHistoryItemOutputDocument, {
    historyItemID,
    runID,
  });

  if (typeof res !== 'object' || res === null || !('functionRun' in res)) {
    throw new Error('invalid response');
  }
  const { functionRun } = res;

  if (
    typeof functionRun !== 'object' ||
    functionRun === null ||
    !('historyItemOutput' in functionRun)
  ) {
    throw new Error('invalid response');
  }
  const { historyItemOutput } = functionRun;

  if (historyItemOutput === null) {
    return undefined;
  }
  if (typeof historyItemOutput !== 'string') {
    throw new Error('invalid response');
  }

  return historyItemOutput;
}
