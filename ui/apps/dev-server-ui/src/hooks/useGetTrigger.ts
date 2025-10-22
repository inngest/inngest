import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import { GetTriggerDocument, type GetTriggerQuery } from '@/store/generated';

export function useGetTrigger() {
  return useCallback(async (runID: string) => {
    const data: GetTriggerQuery = await client.request(GetTriggerDocument, { runID });

    return data.runTrigger;
  }, []);
}
