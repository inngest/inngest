import { notFound } from 'next/navigation';

import { maxRenderedOutputSizeBytes } from '@/app/consts';
import { Alert } from '@/components/Alert';
import SyntaxHighlighter from '@/components/SyntaxHighlighter';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';

export const dynamic = 'force-dynamic';

const GetFunctionRunOutputDocument = graphql(`
  query GetFunctionRunOutput($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        run(id: $functionRunID) {
          output
        }
      }
    }
  }
`);

type FunctionRunOutputProps = {
  params: {
    environmentSlug: string;
    slug: string;
    runId: string;
  };
};

export const runtime = 'nodejs';

export default async function FunctionRunOutput({ params }: FunctionRunOutputProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const environment = await getEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const response = await graphqlAPI.request(GetFunctionRunOutputDocument, {
    environmentID: environment.id,
    functionSlug,
    functionRunID: params.runId,
  });

  const functionRunOutput = response.environment.function?.run.output;

  if (!functionRunOutput) {
    notFound();
  }

  let parsedOutput: string | undefined;
  let isOutputTooLarge = false;
  if (typeof functionRunOutput === 'string') {
    // Keeps the tab from crashing when the output is huge.
    if (functionRunOutput.length > maxRenderedOutputSizeBytes) {
      isOutputTooLarge = true;
    } else {
      try {
        parsedOutput = JSON.stringify(JSON.parse(functionRunOutput), null, 2);
      } catch (error) {
        console.error(`Error parsing JSON output of function: `, error);
        parsedOutput = functionRunOutput;
      }
    }
  }

  if (isOutputTooLarge) {
    return (
      <Alert className="w-fit" severity="warning">
        Output size is too large to render
      </Alert>
    );
  }

  return (
    <div className="p-6">
      {parsedOutput && <SyntaxHighlighter language="json">{parsedOutput}</SyntaxHighlighter>}
    </div>
  );
}
