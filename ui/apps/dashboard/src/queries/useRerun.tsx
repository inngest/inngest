import type {
  RerunPayload,
  RerunResult,
} from "@inngest/components/SharedContext/useRerun";
import { useMutation } from "urql";

import { useEnvironment } from "@/components/Environments/environment-context";
import { graphql } from "@/gql";
import { pathCreator } from "@/utils/urls";

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

export const useRerun = () => {
  const env = useEnvironment();
  const [, mutate] = useMutation(mutation);

  async function rerun({ fnID, runID }: RerunPayload): Promise<RerunResult> {
    try {
      if (!fnID || !runID) {
        throw new Error("envID, fnID, and runID are required");
      }

      const { data, error } = await mutate({
        environmentID: env.id,
        functionID: fnID,
        functionRunID: runID,
      });

      const newRunID = data?.retryWorkflowRun?.id;
      return {
        error,
        data: { newRunID },
        redirect: newRunID
          ? pathCreator.runPopout({ envSlug: env.slug, runID: newRunID })
          : undefined,
      };
    } catch (error) {
      console.error("error rerunning function", error);
      return {
        error:
          error instanceof Error
            ? error
            : new Error("Error re-running function"),
        data: undefined,
      };
    }
  }

  return rerun;
};
