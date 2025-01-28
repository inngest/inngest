import { useCallback } from 'react';
import { Link } from '@inngest/components/Link';
import { toast } from 'sonner';

import { useRerunMutation } from '@/store/generated';
import { pathCreator } from '@/utils/pathCreator';

export function useRerun() {
  const [rerun] = useRerunMutation();

  return useCallback(
    async ({ runID }: { fnID: string; runID: string }) => {
      try {
        const res = await rerun({ runID });
        if ('error' in res) {
          // Throw error so that the modal can catch and display it
          throw res.error;
        }

        const newRunID: unknown = res.data.rerun;
        if (typeof newRunID !== 'string') {
          throw new Error(`invalid run ID: ${newRunID}`);
        }

        // Give user a link to the new run
        toast.success(
          <Link href={pathCreator.runPopout({ runID: newRunID })} target="_blank">
            Successfully queued rerun
          </Link>
        );
      } catch (e) {
        toast.error('Failed to queue rerun');
        throw e;
      }
    },
    [rerun]
  );
}
