import type {
  CancelRunPayload,
  CancelRunResult,
} from '@inngest/components/SharedContext/useCancelRun';
import { useMutation } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const mutation = graphql(`
  mutation CancelRun($envID: UUID!, $runID: ULID!) {
    cancelRun(envID: $envID, runID: $runID) {
      id
    }
  }
`);

export const useCancelRun = () => {
  const env = useEnvironment();
  const [, mutate] = useMutation(mutation);

  async function cancelRun({
    runID,
  }: CancelRunPayload): Promise<CancelRunResult> {
    try {
      return await mutate({ envID: env.id, runID });
    } catch (error) {
      console.error('error cancelling function run', error);
      return {
        error:
          error instanceof Error
            ? error
            : new Error('Error cancelling function run'),
        data: undefined,
      };
    }
  }

  return cancelRun;
};
