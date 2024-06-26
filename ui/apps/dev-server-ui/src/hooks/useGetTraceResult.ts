import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import { GetTraceResultDocument, type GetTraceResultQuery } from '@/store/generated';

export function useGetTraceResult() {
  return useCallback(async (runID: string) => {
    const data: GetTraceResultQuery = await client.request(GetTraceResultDocument, { runID });

    return data.runTraceSpanOutputByID;
  }, []);
}
