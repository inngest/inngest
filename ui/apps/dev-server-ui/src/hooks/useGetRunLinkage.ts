import { useCallback } from 'react';
import type {
  GetRunLinkagePayload,
  GetRunLinkageResult,
} from '@inngest/components/SharedContext/useGetRunLinkage';

import { client } from '@/store/baseApi';
import {
  GetRunLinkageDocument,
  type GetRunLinkageQuery,
} from '@/store/generated';

export function useGetRunLinkage() {
  return useCallback(
    async ({ runID }: GetRunLinkagePayload): Promise<GetRunLinkageResult> => {
      const data: GetRunLinkageQuery = await client.request(
        GetRunLinkageDocument,
        {
          runID,
        },
      );
      const run = data.run;
      if (!run) {
        throw new Error('missing run');
      }
      return {
        loading: false,
        data: {
          defers: run.defers,
          siblingDefers: run.siblingDefers,
          deferredFrom: run.deferredFrom,
        },
      };
    },
    [],
  );
}
