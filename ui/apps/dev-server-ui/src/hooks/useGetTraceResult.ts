import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import { GetTraceResultDocument, type GetTraceResultQuery } from '@/store/generated';

export function useGetTraceResult() {
  return useCallback(async (traceID: string, preview?: boolean) => {
    const data: GetTraceResultQuery = await client.request(GetTraceResultDocument, {
      traceID,
      preview,
    });

    return data.runTraceSpanOutputByID;
  }, []);
}
