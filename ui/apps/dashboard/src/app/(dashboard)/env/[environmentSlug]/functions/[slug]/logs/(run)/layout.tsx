'use client';

import { type Route } from 'next';
import { useParams, useRouter } from 'next/navigation';
import { SlideOver } from '@inngest/components/SlideOver';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';

type RunLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    slug: string;
  };
};

const GetFunctionRunTriggersDocument = graphql(`
  query GetFunctionRunTriggers($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        run(id: $functionRunID) {
          id
          version: workflowVersion {
            triggers {
              schedule
            }
          }
        }
      }
    }
  }
`);

export default function RunLayout({ children, params }: RunLayoutProps) {
  const router = useRouter();
  const { runId } = useParams();
  const functionRunID = typeof runId === 'string' ? runId : '';

  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const [{ data, fetching: fetchingRunTriggers }] = useQuery({
    query: GetFunctionRunTriggersDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug: params.slug,
      functionRunID,
    },
    pause: !environment?.id,
  });

  const triggers = data?.environment?.function?.run?.version?.triggers;
  const hasCron =
    Array.isArray(triggers) && triggers.length > 0 && triggers.some((trigger) => trigger.schedule);

  if (isFetchingEnvironment || fetchingRunTriggers) {
    return;
  }

  return (
    <SlideOver
      size={hasCron ? 'small' : 'large'}
      onClose={() =>
        router.push(
          `/env/${params.environmentSlug}/functions/${encodeURIComponent(
            params.slug
          )}/logs` as Route
        )
      }
    >
      {children}
    </SlideOver>
  );
}
