import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { CancellationTable } from './CancellationTable';

const query = graphql(`
  query GetFnCancellations($envSlug: String!, $fnSlug: String!) {
    env: envBySlug(slug: $envSlug) {
      fn: workflowBySlug(slug: $fnSlug) {
        cancellations {
          edges {
            node {
              createdAt
              id
              queuedAtMax
              queuedAtMin
            }
          }
        }
      }
    }
  }
`);

type Props = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default async function Page({ params }: Props) {
  const envSlug = decodeURIComponent(params.environmentSlug);
  const fnSlug = decodeURIComponent(params.slug);

  // TODO: Add pagination
  const res = await graphqlAPI.request(query, {
    envSlug,
    fnSlug,
  });

  if (!res.env) {
    throw new Error('environment not found');
  }
  if (!res.env.fn) {
    throw new Error('function not found');
  }
  const data = res.env.fn.cancellations.edges.map((edge) => edge.node);

  return <CancellationTable data={data} />;
}
