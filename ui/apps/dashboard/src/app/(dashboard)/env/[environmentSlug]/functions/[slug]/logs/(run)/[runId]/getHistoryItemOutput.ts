import { Client } from 'urql';

import { graphql } from '@/gql';

const getHistoryItemOutputDocument = graphql(`
  query GetHistoryItemOutput($envID: ID!, $functionID: ID!, $historyItemID: ULID!, $runID: ULID!) {
    environment: workspace(id: $envID) {
      function: workflow(id: $functionID) {
        run(id: $runID) {
          historyItemOutput(id: $historyItemID)
        }
      }
    }
  }
`);

export async function getHistoryItemOutput({
  client,
  envID,
  functionID,
  historyItemID,
  runID,
}: {
  client: Client;
  envID: string;
  functionID: string;
  historyItemID: string;
  runID: string;
}): Promise<string | undefined> {
  // TODO: How to get type annotations? It returns `any`.
  const res = await client
    .query(getHistoryItemOutputDocument, {
      envID,
      functionID,
      historyItemID,
      runID,
    })
    .toPromise();
  if (res.error) {
    throw res.error;
  }

  return res.data?.environment.function?.run.historyItemOutput ?? undefined;
}
