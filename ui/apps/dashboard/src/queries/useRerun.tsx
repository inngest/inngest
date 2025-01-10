import { Link } from '@inngest/components/Link';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

const mutation = graphql(`
  mutation RerunFunctionRun($environmentID: ID!, $functionID: ID!, $functionRunID: ULID!) {
    retryWorkflowRun(
      input: { workspaceID: $environmentID, workflowID: $functionID }
      workflowRunID: $functionRunID
    ) {
      id
    }
  }
`);

export function useRerun({ envID, envSlug }: { envID: string; envSlug: string }) {
  const [, mutate] = useMutation(mutation);

  async function rerun({ fnID, runID }: { fnID: string; runID: string }): Promise<void> {
    try {
      const response = await mutate({
        environmentID: envID,
        functionID: fnID,
        functionRunID: runID,
      });
      if (response.error) {
        throw response.error;
      }
      const newRunID = response.data?.retryWorkflowRun?.id;
      if (!newRunID) {
        throw new Error('missing new run ID');
      }

      // Give user a link to the new run
      toast.success(
        <Link href={pathCreator.runPopout({ envSlug, runID: newRunID })} target="_blank">
          Successfully queued rerun
        </Link>
      );
    } catch (e) {
      toast.error('Failed to queue rerun');
      throw e;
    }
  }

  return rerun;
}
