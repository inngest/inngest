import { notFound } from 'next/navigation';

import SyntaxHighlighter from '@/components/SyntaxHighlighter';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';

export const dynamic = 'force-dynamic';

const GetFunctionRunPayloadDocument = graphql(`
  query GetFunctionRunPayload($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        functionRun: run(id: $functionRunID) {
          events {
            payload: event
          }
        }
      }
    }
  }
`);

type FunctionRunPayloadProps = {
  params: {
    environmentSlug: string;
    slug: string;
    runId: string;
  };
};

export const runtime = 'nodejs';

export default async function FunctionRunPayload({ params }: FunctionRunPayloadProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const environment = await getEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const response = await graphqlAPI.request(GetFunctionRunPayloadDocument, {
    environmentID: environment.id,
    functionSlug,
    functionRunID: params.runId,
  });

  const functionRunPayload = response.environment.function?.functionRun.events.map((evt) =>
    JSON.parse(evt.payload)
  );

  if (!functionRunPayload || functionRunPayload.length === 0) {
    notFound();
  }

  const payload = functionRunPayload.length === 1 ? functionRunPayload[0] : functionRunPayload;
  const formattedFunctionRunPayload = JSON.stringify(payload, null, 2);

  return (
    <div className="p-6">
      <SyntaxHighlighter language="json">{formattedFunctionRunPayload}</SyntaxHighlighter>
    </div>
  );
}
